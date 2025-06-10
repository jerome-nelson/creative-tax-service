package main

import (
	"bytes"
	"context"
	"encoding/base64"
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
	"github.com/rs/cors"
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

type Oauth struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	ExpiresIn   int    `json:"expires_in"`
	// Requires the scope `offline_access` to be set
	RefreshToken string `json:"refresh_token"`
}

type SearchResults struct {
	Issues []any          `json:"issues"`
	Names  map[string]any `json:"names"`
	Cursor string         `json:"nextPageToken"`
	Schema map[string]any `json:"string"`
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

func handleJiraSearch(log *log.Logger, config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.URL.Query().Get("user")

		if email == "" {
			http.Error(w, "Invalid user credentials", http.StatusBadRequest)
		}

		// Need to fetch the Jira instance url:
		// https://activecampaign.atlassian.net/
		bodyData := `{
		  "expand": "names, changelog",
		  "fields": [
			"*all"
		  ],
		  "fieldsByKeys": true,
		  "jql": "assignee = currentUser() AND statusCategoryChangedDate >= \"2025-05-01\" AND statusCategoryChangedDate <= \"2025-05-30\" ORDER BY statusCategoryChangedDate DESC"
		}`

		cookie, err := r.Cookie("oauth_token")
		if err != nil {
			switch {
			case errors.Is(err, http.ErrNoCookie):
				http.Error(w, "cookie not found", http.StatusBadRequest)
			default:
				log.Println(err)
				http.Error(w, "server error", http.StatusInternalServerError)
			}
			return
		}

		var token = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", email, cookie.Value)))
		request, err := http.NewRequest("POST", "https://activecampaign.atlassian.net/rest/api/3/search/jql", bytes.NewBuffer([]byte(bodyData)))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Authorization", "Basic "+token)

		client := &http.Client{}
		results, err := client.Do(request)

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Println("Error closing body:", err)
			}
		}(results.Body)

		if results != nil {
			if results.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(results.Body)
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println(string(body))
				return
			}

			res := SearchResults{}
			err := json.NewDecoder(results.Body).Decode(&res)
			if err != nil {
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println("Error retrieving oauth:", err)
				return
			}

			if err := encode(w, r, http.StatusOK, res); err != nil {
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

func handleIssue(log *log.Logger, config *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.URL.Query().Get("user")

		if email == "" {
			http.Error(w, "Invalid user credentials", http.StatusBadRequest)
		}

		// Need to fetch the Jira instance url:
		// https://activecampaign.atlassian.net/
		bodyData := `{
		  "expand": "names, changelog",
		  "fields": [
			"*all"
		  ],
		  "fieldsByKeys": true,
		  "jql": "assignee = currentUser() AND statusCategoryChangedDate >= \"2025-05-01\" AND statusCategoryChangedDate <= \"2025-05-30\" ORDER BY statusCategoryChangedDate DESC"
		}`

		cookie, err := r.Cookie("oauth_token")
		if err != nil {
			switch {
			case errors.Is(err, http.ErrNoCookie):
				http.Error(w, "cookie not found", http.StatusBadRequest)
			default:
				log.Println(err)
				http.Error(w, "server error", http.StatusInternalServerError)
			}
			return
		}

		var token = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", email, cookie.Value)))
		request, err := http.NewRequest("GET", "https://activecampaign.atlassian.net/rest/api/3/issue/FOR2-137?fields=*all", bytes.NewBuffer([]byte(bodyData)))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Accept", "application/json")
		request.Header.Set("Authorization", "Basic "+token)

		client := &http.Client{}
		results, err := client.Do(request)

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Println("Error closing body:", err)
			}
		}(results.Body)

		if results != nil {
			if results.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(results.Body)
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println(string(body))
				return
			}

			res := SearchResults{}
			err := json.NewDecoder(results.Body).Decode(&res)
			if err != nil {
				http.Error(w, "Error Authenticating", http.StatusBadRequest)
				log.Println("Error retrieving oauth:", err)
				return
			}

			if err := encode(w, r, http.StatusOK, res); err != nil {
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

func addRoutes(mux *http.ServeMux, config *Config, log *log.Logger) {

	allowMethod := methodGuard(log)
	hasAuth := authGuard(log)

	mux.HandleFunc("/single", allowMethod(http.MethodPost, hasAuth(handleIssue(log, config))))
	mux.HandleFunc("/search", allowMethod(http.MethodPost, hasAuth(handleJiraSearch(log, config))))
	mux.HandleFunc("/health", allowMethod(http.MethodGet, handleHealthCheck(log)))
	mux.HandleFunc("/refresh", allowMethod(http.MethodPost, handleRefreshToken(log, config)))
	mux.HandleFunc("/oauth", allowMethod(http.MethodPost, handleGenerateToken(log, config)))
}

func ServerInstance(config *Config, log *log.Logger) http.Handler {
	mux := http.NewServeMux()
	var handler http.Handler = mux
	addRoutes(mux, config, log)
	// Make ALLOWED ORIGINS an .env array
	// Same with Allowed Headers ..mb
	handler = cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5000", "http://localhost:7000"},
		AllowCredentials: true,
		Debug:            true,
		AllowedHeaders:   []string{"X-Code", "Set-Cookie"},
	}).Handler(mux)
	return handler
}

func GetConfig() *Config {
	err := godotenv.Load("jira.env")
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
		RedirectUrl: os.Getenv("REDIRECT_URI"),
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
