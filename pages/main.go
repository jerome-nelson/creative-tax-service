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

	"github.com/joho/godotenv"
)

//ToStudy
// Context
// defer
// Difference between := and var
// * & - what do these mean?
// get used to writing anon functions
// What is a rune?

type Page struct {
	Title   string
	Message string
}

type Config struct {
	shared.ServerConfig
	shared.JiraConfig
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

		data := Page{
			Title:   "Zend",
			Message: shared.SetAuthUrl(config.JiraConfig),
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

func handleStaticFiles(log *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("attempted to access: " + r.URL.Path)
		http.ServeFile(w, r, r.URL.Path)
	}
}

func addRoutes(mux *http.ServeMux, config *Config, log *log.Logger) {

	allowMethod := shared.MethodGuard(log)

	// Add a template system for index routes
	// Serve 404 if template not found
	// Nice to have: Move templates and static into pages folder
	mux.Handle("/static/", http.StripPrefix("/", allowMethod(http.MethodGet, handleStaticFiles(log))))
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
	err := godotenv.Load("pages.env")
	if err != nil {
		log.Fatal("Error loading env variables")
	}
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

	// Look into hoisting these higher?
	config := GetConfig()
	logger := log.New(os.Stdout, "["+config.ServiceName+"] ", log.LstdFlags)

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
