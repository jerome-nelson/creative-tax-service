package main

import (
	"JiraConnect/shared"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

//ToStudy
// Context
// defer
// Difference between := and var
// * & - what do these mean?
// get used to writing anon functions
// What is a rune?

type Config struct {
	shared.ServerConfig
	shared.JiraConfig
}

// TODO: Add a generic grant handler for both refresh and auth
func handleGenerateToken(log *log.Logger, config shared.JiraConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.Header.Get("X-Code")
		if code == "" {
			http.Error(w, "Missing X-Code", http.StatusBadRequest)
		}
		values := map[string]string{
			"grant_type":    "authorization_code",
			"code":          code,
			"client_id":     config.Cid,
			"client_secret": config.Secret,
			"redirect_uri":  config.RedirectUrl,
		}

		jsonValue, _ := json.Marshal(values)
		oauth, err := http.Post(
			config.OauthUrl,
			"application/json",
			bytes.NewBuffer(jsonValue),
		)

		if oauth != nil {
			if oauth.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(oauth.Body)
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println(string(body))
				return
			}

			res := shared.Oauth{}
			err := json.NewDecoder(oauth.Body).Decode(&res)
			if err != nil {
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println("oauth error:", err)
				return
			}

			if res.AccessToken == "" {
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println("Error retrieving oauth - JSON is empty")
			}

			shared.SetJiraCookie(w, log, res)
		}

		if err != nil {
			http.Error(w, "Error retrieving authentication", http.StatusInternalServerError)
			log.Println("Error retrieving oauth:", err)
			return
		}

		defer oauth.Body.Close()
	}
}

// TODO: Refactor both JIRA auth calls to allow the calling of one function for both
func handleRefreshToken(log *log.Logger, config shared.JiraConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var token string
		if token = r.Header.Get("x-refresh"); token == "" {
			http.Error(w, "Unauthorised", http.StatusUnauthorized)
		}

		values := map[string]string{
			"grant_type":    "refresh_token",
			"refresh_token": token,
			"client_id":     config.Cid,
			"client_secret": config.Secret,
		}

		jsonValue, _ := json.Marshal(values)
		oauth, err := http.Post(
			config.OauthUrl,
			"application/json",
			bytes.NewBuffer(jsonValue),
		)

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Println("Error closing body:", err)
			}
		}(oauth.Body)

		if oauth != nil {
			if oauth.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(oauth.Body)
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println(string(body))
				return
			}

			res := shared.Oauth{}
			err := json.NewDecoder(oauth.Body).Decode(&res)
			if err != nil {
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println("failed to retrieve refresh: ", err)
				return
			}

			if res.AccessToken == "" {
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println("failed to retrieve refresh - JSON is empty")
			}

			shared.SetJiraCookie(w, log, res)
		}

		if err != nil {
			http.Error(w, "Error retrieving authentication", http.StatusInternalServerError)
			log.Println("Error retrieving oauth:", err)
			return
		}
	}

}

func handleTempIssue(log *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var anyJson map[string]interface{}
		jsonFile, err := os.ReadFile("./jira/issues-1-sample.json")
		err2 := json.Unmarshal(jsonFile, &anyJson)
		if err != nil || err2 != nil {
			if err != nil {
				fmt.Println(err)
			}
			if err2 != nil {
				fmt.Println(err2)
			}
			http.Error(w, "Error retrieving file", http.StatusInternalServerError)
			return
		}

		log.Println("Successfully Opened ./jira/issues-1-sample.json")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(anyJson)
	}
}

func addRoutes(mux *http.ServeMux, config *Config, log *log.Logger) {

	allowMethod := shared.MethodGuard(log)

	mux.HandleFunc("/health", allowMethod(http.MethodGet, shared.HandleHealthCheck(log)))
	mux.HandleFunc("/refresh", allowMethod(http.MethodPost, handleRefreshToken(log, config.JiraConfig)))
	mux.HandleFunc("/oauth", allowMethod(http.MethodPost, handleGenerateToken(log, config.JiraConfig)))
	mux.Handle("/temp", http.StripPrefix("/", allowMethod(http.MethodGet, handleTempIssue(log))))
}

func ServerInstance(config *Config, log *log.Logger) http.Handler {
	mux := http.NewServeMux()
	var handler http.Handler = mux
	addRoutes(mux, config, log)
	handler = shared.HandleCors(mux, log)
	return handler
}

func GetConfig() *Config {
	err := godotenv.Load("jira.env")
	if err != nil {
		log.Fatal("Error loading env variables")
	}
	return &Config{
		JiraConfig: shared.JiraConfig{
			RedirectUrl: os.Getenv("REDIRECT_URL"),
			Cid:         os.Getenv("CLIENT_ID"),
			Secret:      os.Getenv("CLIENT_SECRET"),
			OauthUrl:    os.Getenv("OAUTH_URL"),
		},
		ServerConfig: shared.ServerConfig{
			Port:        os.Getenv("PORT"),
			Host:        os.Getenv("HOST"),
			ServiceName: os.Getenv("SERVICE_NAME"),
		},
	}
}

func run(ctx context.Context) error {

	// Split these into other files that get included into the application
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
