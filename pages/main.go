package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// TODO Needs CORS
//ToStudy
// Context
// defer
// Difference between := and var
// * & - what do these mean?
// get used to writing anon functions
// What is a rune?

type Config struct {
	Port        string
	Host        string
	Cid         string
	ServiceName string
	RedirectUrl string
}

type Page struct {
	Title   string
	Message string
}

type Oauth struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	ExpiresIn   int    `json:"expires_in"`
	// Requires the scope `offline_access` to be set
	RefreshToken string `json:"refresh_token"`
}

func authGuard(log *log.Logger) func(h http.HandlerFunc) http.HandlerFunc {
	log.Println("authGuard init")
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var isAuthed bool = false
			for _, cookie := range r.Cookies() {
				if cookie.Name == "oauth_token" {
					isAuthed = true
				}
			}
			if isAuthed != true {
				log.Printf("attempted to access auth route %s\n", r.URL.Path)
				http.Error(w, "Not authorised", http.StatusUnauthorized)
				return
			}

			h(w, r)
		}
	}
}

func methodGuard(log *log.Logger) func(method string, h http.HandlerFunc) http.HandlerFunc {
	log.Println("methodGuard init")
	return func(method string, h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != method {
				log.Printf("method %s attempted on %s\n", r.Method, r.URL.Path)
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
				return
			}

			h(w, r)
		}
	}
}

func encode[T any](w http.ResponseWriter, _ *http.Request, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

func handleHealthCheck(log *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tm := time.Now().Format(time.RFC1123)
		data := map[string]any{
			"result": "Pong at " + tm,
		}

		if err := encode(w, r, http.StatusOK, data); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			log.Println(err)
		}
	}
}

func handleAuth(log *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		tmpl, err := template.ParseFiles("templates/auth.html")
		if err != nil {
			http.Error(w, "Error parsing template", http.StatusInternalServerError)
			log.Println("Template parsing error:", err)
			return
		}

		data := Page{
			Title:   "Auth Page",
			Message: "This is the auth page. You will be redirected back to home",
		}

		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, "Error executing template", http.StatusInternalServerError)
			log.Println("Template execution error:", err)
		}
	}

}

func handleRoot(log *log.Logger, config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		scopes := []string{
			"offline_access",
			"read:me",
			"read:project.avatar:jira",
			"read:filter:jira",
			"read:group:jira",
			"read:issue:jira",
			"read:attachment:jira",
			"read:comment:jira",
			"read:comment.property:jira",
			"read:field:jira",
		}

		baseURL := "https://auth.atlassian.com/authorize"
		params := url.Values{}
		params.Set("audience", "api.atlassian.com")
		params.Set("client_id", config.Cid) // fill your actual client_id
		params.Set("redirect_uri", config.RedirectUrl)
		params.Set("response_type", "code")
		params.Set("prompt", "consent")

		params.Set("scope", strings.Join(scopes, " "))

		data := Page{
			Title:   "Creative Tax Generator",
			Message: fmt.Sprintf("%s?%s", baseURL, params.Encode()),
		}
		tmpl, err := template.ParseFiles("templates/index.html")
		if err != nil {
			http.Error(w, "Error parsing template", http.StatusInternalServerError)
			log.Println("root template error", err)
			return
		}

		if err = tmpl.Execute(w, data); err != nil {
			http.Error(w, "Error executing template", http.StatusInternalServerError)
			log.Println("error applying template", err)
		}
	}
}

func addRoutes(mux *http.ServeMux, config *Config, log *log.Logger) {

	allowMethod := methodGuard(log)
	//hasAuth := authGuard(log)

	mux.HandleFunc("/health", allowMethod(http.MethodGet, handleHealthCheck(log)))
	mux.HandleFunc("/auth", allowMethod(http.MethodGet, handleAuth(log)))
	mux.HandleFunc("/", allowMethod(http.MethodGet, handleRoot(log, config)))
}

func ServerInstance(config *Config, log *log.Logger) http.Handler {
	mux := http.NewServeMux()
	var handler http.Handler = mux
	addRoutes(mux, config, log)
	return handler
}

func GetConfig() *Config {
	err := godotenv.Load("pages.env")
	if err != nil {
		log.Fatal("Error loading env variables")
	}
	return &Config{
		Cid:         os.Getenv("CLIENT_ID"),
		Port:        os.Getenv("PORT"),
		Host:        os.Getenv("HOST"),
		ServiceName: os.Getenv("SERVICE_NAME"),
		RedirectUrl: "http://" + os.Getenv("HOST") + ":" + os.Getenv("PORT") + "/auth",
	}
}

func run(ctx context.Context) error {

	// Look into hoisting these higher?
	config := GetConfig()
	logger := log.New(os.Stdout, config.ServiceName+": ", log.Lmsgprefix)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	srv := ServerInstance(config, logger)
	httpServer := &http.Server{
		Addr:    net.JoinHostPort(config.Host, config.Port),
		Handler: srv,
	}

	go func() {
		logger.Printf("service started - %s\n", httpServer.Addr)
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
