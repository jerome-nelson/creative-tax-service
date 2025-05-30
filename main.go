package main

import (
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

//ToStudy
// Context
// defer
// Difference between := and var
// * & - what do these mean?

type Config struct {
	Port string
	Host string
}

type Page struct {
	Title string
	Body  *template.Template
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	tm := time.Now().Format(time.RFC1123)
	w.Write([]byte("Pong at: " + tm))
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		log.Println("Template parsing error:", err)
		return
	}

	data := struct {
		Title   string
		Message string
	}{
		Title:   "Welcome Page",
		Message: "Hello, Go Templates!",
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		log.Println("Template execution error:", err)
	}
}

func addRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", handleHealthCheck)
	mux.HandleFunc("/", handleRoot)
}

func ServerInstance() http.Handler {
	mux := http.NewServeMux()
	var handler http.Handler = mux
	addRoutes(mux)
	return handler
}

func run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	config := &Config{
		Port: "5000",
		Host: "localhost",
	}
	srv := ServerInstance()
	httpServer := &http.Server{
		Addr:    net.JoinHostPort(config.Host, config.Port),
		Handler: srv,
	}

	go func() {
		log.Printf("listening on %s\n", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
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
			fmt.Fprintf(os.Stderr, "error shutting down http server: %s\n", err)
		}
	}()
	wg.Wait()
	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
