package shared

import (
	"encoding/json"
	"fmt"
	"github.com/rs/cors"
	"log"
	"net/http"
	"path"
	"strings"
	"time"
)

type ServerConfig struct {
	Port        string
	Host        string
	ServiceName string
}

// TODO: Allow options to be overridden/merged if needed
func HandleCors(mux *http.ServeMux, log *log.Logger) http.Handler {
	log.Println("cors initialised")
	return cors.New(cors.Options{
		AllowCredentials: true,
		Debug:            false,
		// For one service only - jira/main.go
		// Make ALLOWED ORIGINS an .env array
		// Same with Allowed Headers ..mb
		AllowedOrigins: []string{"http://localhost:5000", "http://localhost:7000"},
		AllowedHeaders: []string{"X-Code", "Set-Cookie"},
	}).Handler(mux)
}

func Encode[T any](w http.ResponseWriter, status int, v T) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

func MethodGuard(log *log.Logger) func(method string, h http.HandlerFunc) http.HandlerFunc {
	log.Println("method guard initialised")
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

func RestrictExtensions(log *log.Logger, allowed map[string]bool) func(h http.HandlerFunc) http.HandlerFunc {
	log.Println("restrict extensions initialised")
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ext := strings.ToLower(path.Ext(r.URL.Path))

			if !allowed[ext] {
				log.Printf("client requested forbidden file type: %s", r.URL.Path)
				http.Error(w, "Server Error", http.StatusForbidden)
				return
			}

			h(w, r)
		}
	}
}

func HandleHealthCheck(log *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tm := time.Now().Format(time.RFC1123)
		data := map[string]any{
			"result": "Pong at " + tm,
		}

		if err := Encode(w, http.StatusOK, data); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			log.Println(err)
		}
	}
}
