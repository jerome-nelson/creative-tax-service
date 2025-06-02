package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
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
	Csecrt      string
	OauthUrl    string
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

// OPTIONS
func handleGenerateToken(log *log.Logger, config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.Header.Get("X-Code")
		if code == "" {
			http.Error(w, "Missing X-Code", http.StatusBadRequest)
		}
		values := map[string]string{
			"grant_type":    "authorization_code",
			"code":          code,
			"client_id":     config.Cid,
			"client_secret": config.Csecrt,
			"redirect_uri":  config.RedirectUrl,
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

			res := Oauth{}
			err := json.NewDecoder(oauth.Body).Decode(&res)
			if err != nil {
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println("Error retrieving oauth:", err)
				return
			}

			if res.AccessToken == "" {
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println("Error retrieving oauth - JSON is empty")
			}

			offsetExpiry := res.ExpiresIn - 60

			http.SetCookie(w, &http.Cookie{
				Name:     "oauth_token",
				Value:    res.AccessToken,
				Path:     "/",
				MaxAge:   offsetExpiry,
				HttpOnly: false,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			})
			http.SetCookie(w, &http.Cookie{
				Name:     "scopes",
				Value:    res.Scope,
				Path:     "/",
				MaxAge:   offsetExpiry,
				HttpOnly: false,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			})

			http.SetCookie(w, &http.Cookie{
				Name:     "refresh_token",
				Value:    res.RefreshToken,
				Path:     "/",
				MaxAge:   offsetExpiry,
				HttpOnly: false,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			})

			now := time.Now().UTC()
			futureTime := now.Add(time.Duration(offsetExpiry) * time.Second)

			http.SetCookie(w, &http.Cookie{
				Name:     "expiry",
				Value:    futureTime.Format("2006-01-02T15:04:05.000Z"),
				Path:     "/",
				MaxAge:   offsetExpiry,
				HttpOnly: false,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			})
			if err := encode(w, r, http.StatusOK, map[string]string{
				"scope": res.Scope,
				"type":  res.TokenType,
			}); err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				log.Println(err)
			}
		}

		if err != nil {
			http.Error(w, "Error retrieving authentication", http.StatusInternalServerError)
			log.Println("Error retrieving oauth:", err)
			return
		}
	}

}

// TODO: Refactor both JIRA auth calls to combine the setCookie Logic and calls
func handleRefreshToken(log *log.Logger, config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var token string
		if token = r.Header.Get("x-refresh"); token == "" {
			http.Error(w, "Unauthorised", http.StatusUnauthorized)
		}

		values := map[string]string{
			"grant_type":    "refresh_token",
			"refresh_token": token,
			"client_id":     config.Cid,
			"client_secret": config.Csecrt,
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

			res := Oauth{}
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

			offsetExpiry := res.ExpiresIn - 60

			http.SetCookie(w, &http.Cookie{
				Name:     "oauth_token",
				Value:    res.AccessToken,
				Path:     "/",
				MaxAge:   offsetExpiry,
				HttpOnly: false,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			})

			http.SetCookie(w, &http.Cookie{
				Name:     "scopes",
				Value:    res.Scope,
				Path:     "/",
				MaxAge:   offsetExpiry,
				HttpOnly: false,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			})

			http.SetCookie(w, &http.Cookie{
				Name:     "refresh_token",
				Value:    res.RefreshToken,
				Path:     "/",
				MaxAge:   offsetExpiry,
				HttpOnly: false,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			})

			now := time.Now().UTC()
			futureTime := now.Add(time.Duration(offsetExpiry) * time.Second)

			http.SetCookie(w, &http.Cookie{
				Name:     "expiry",
				Value:    futureTime.Format("2006-01-02T15:04:05.000Z"),
				Path:     "/",
				MaxAge:   offsetExpiry,
				HttpOnly: false,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			})
			if err := encode(w, r, http.StatusOK, map[string]string{
				"scope": res.Scope,
				"type":  res.TokenType,
			}); err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				log.Println(err)
			}
		}

		if err != nil {
			http.Error(w, "Error retrieving authentication", http.StatusInternalServerError)
			log.Println("Error retrieving oauth:", err)
			return
		}
	}

}

func handleRoot(log *log.Logger, config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		scopes := []string{"offline_access", "read:me"}
		scopeParam := strings.Join(scopes, " ")
		encodedScopes := url.QueryEscape(scopeParam)
		encodedRedirectURL := url.QueryEscape(config.RedirectUrl)

		data := Page{
			Title: "Creative Tax Generator",
			Message: "https://auth.atlassian.com/authorize?audience=api.atlassian.com" +
				"&client_id=" + config.Cid +
				"&scope=" + encodedScopes +
				"&redirect_uri=" + encodedRedirectURL +
				"&response_type=code&prompt=consent",
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
	mux.HandleFunc("/refresh", allowMethod(http.MethodPost, handleRefreshToken(log, config)))
	mux.HandleFunc("/oauth", allowMethod(http.MethodPost, handleGenerateToken(log, config)))
	mux.HandleFunc("/", allowMethod(http.MethodGet, handleRoot(log, config)))
}

func ServerInstance(config *Config, log *log.Logger) http.Handler {
	mux := http.NewServeMux()
	var handler http.Handler = mux
	addRoutes(mux, config, log)
	return handler
}

func GetConfig(log *log.Logger) *Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading env variables")
	}
	return &Config{
		Port:        os.Getenv("PORT"),
		Host:        os.Getenv("HOST"),
		OauthUrl:    os.Getenv("OAUTH_URL"),
		Cid:         os.Getenv("CLIENT_ID"),
		Csecrt:      os.Getenv("CLIENT_SECRET"),
		ServiceName: os.Getenv("SERVICE_NAME"),
		RedirectUrl: "http://" + os.Getenv("HOST") + ":" + os.Getenv("PORT") + "/auth",
	}
}

func run(ctx context.Context) error {

	// Look into hoisting these higher?
	logger := log.New(os.Stdout, "jira-auth: ", log.Lmsgprefix)
	config := GetConfig(logger)

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
