package main

import (
	"JiraConnect/shared"
	"context"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

type Page struct {
	Title     string
	Message   string
	ScriptUrl template.JS
}

type Config struct {
	shared.ServerConfig
	shared.JiraConfig
}

func addRoutes(mux *http.ServeMux, config *Config, log *log.Logger) {

	allowMethod := shared.MethodGuard(log)
	webTypesWhitelist := allowWebTypesOnly(log)

	mux.Handle("/static/", http.StripPrefix("/", allowMethod(http.MethodGet, webTypesWhitelist(handleStaticFiles(log)))))
	mux.HandleFunc("/health", allowMethod(http.MethodGet, shared.HandleHealthCheck(log)))
	mux.HandleFunc("/auth", allowMethod(http.MethodGet, handleAuth(log)))
	mux.HandleFunc("/", allowMethod(http.MethodGet, handleRoot(log, config)))
}

func ServerInstance(config *Config, log *log.Logger) http.Handler {
	mux := http.NewServeMux()
	var handler http.Handler = mux
	handler = shared.HandleCors(mux, log)
	// Dont like the pattern difference. Fix
	addRoutes(mux, config, log)
	return handler
}

func GetConfig() *Config {
	return &Config{
		JiraConfig: shared.JiraConfig{
			RedirectUrl: os.Getenv("REDIRECT_URL"),
			Cid:         os.Getenv("CLIENT_ID"),
		},
		ServerConfig: shared.ServerConfig{
			Port:        os.Getenv("PORT"),
			Host:        os.Getenv("HOST"),
			ServiceName: os.Getenv("SERVICE_NAME"),
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
				logger.Printf("error listening and serving: %s", err)
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
