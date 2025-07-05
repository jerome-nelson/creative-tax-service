package main

import (
	"JiraConnect/shared"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Config struct {
	shared.ServerConfig
	shared.JiraConfig
	LLMConfig
}

func addRoutes(mux *http.ServeMux, config *Config, log *log.Logger) {
	allowMethod := shared.MethodGuard(log)
	authGuard := shared.AuthGuard(log)

	mux.HandleFunc("/health", allowMethod(http.MethodGet, shared.HandleHealthCheck(log)))
	mux.HandleFunc("/refresh", allowMethod(http.MethodPost, handleRefreshToken(log, config.JiraConfig)))
	mux.HandleFunc("/oauth", allowMethod(http.MethodPost, handleGenerateToken(log, config.JiraConfig)))
	mux.HandleFunc("/transform", allowMethod(http.MethodPost, authGuard(handlePartiallyGeneratedIssueTransform(log, config.LLMConfig))))
	mux.Handle("/temp", http.StripPrefix("/", allowMethod(http.MethodGet, handleTempIssue(log))))
}

func ServerInstance(config *Config, log *log.Logger) http.Handler {
	mux := http.NewServeMux()
	var handler http.Handler = mux
	addRoutes(mux, config, log)
	handler = shared.HandleCors(mux, log, config.ServerConfig)
	return handler
}

func GetConfig() *Config {
	return &Config{
		JiraConfig: shared.JiraConfig{
			RedirectUrl: os.Getenv("REDIRECT_URL"),
			Cid:         os.Getenv("CLIENT_ID"),
			Secret:      os.Getenv("CLIENT_SECRET"),
			OauthUrl:    os.Getenv("OAUTH_URL"),
		},
		ServerConfig: shared.ServerConfig{
			Port:           os.Getenv("PORT"),
			Host:           os.Getenv("HOST"),
			ServiceName:    os.Getenv("SERVICE_NAME"),
			AllowedOrigins: strings.Split(os.Getenv("ALLOWED_ORIGINS"), ","),
			AllowedHeaders: strings.Split(os.Getenv("ALLOWED_HEADERS"), ","),
		},
		LLMConfig: LLMConfig{
			ApiKey: os.Getenv("LLM_API_KEY"),
		},
	}
}

func run(ctx context.Context) error {
	config := GetConfig()
	logger := log.New(os.Stdout, "["+config.ServiceName+"] ", log.LstdFlags|log.Lshortfile)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	srv := ServerInstance(config, logger)
	httpServer := &http.Server{
		Addr:    net.JoinHostPort(config.Host, config.Port),
		Handler: srv,
	}

	go func() {
		logger.Printf("server started - %s\n", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			_, err := fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
			if err != nil {
				return
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx := context.Background()
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			_, err := fmt.Fprintf(os.Stderr, "error shutting down http server: %s\n", err)
			if err != nil {
				return
			}
		}
	}()
	wg.Wait()
	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "%s\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}
