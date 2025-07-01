package main

import (
	"JiraConnect/shared"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func handleAuth(log *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		tmpl, err := template.ParseFiles("pages/templates/auth.html")
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
			Title:     "Zend",
			ScriptUrl: template.JS(shared.SetAuthUrl(config.JiraConfig)),
		}
		tmpl, err := template.ParseFiles("pages/templates/index.html")
		if err != nil {
			http.Error(w, "Error parsing template", http.StatusInternalServerError)
			log.Println("root template error", err)
			return
		}

		// TODO: Review why ScriptUrl is being double escaped when it doesn't needed to be
		if err = tmpl.Execute(w, data); err != nil {
			http.Error(w, "Error executing template", http.StatusInternalServerError)
			log.Println("error applying template", err)
		}
	}
}

func handleStaticFiles(log *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestedPath := filepath.Join("pages", r.URL.Path)
		log.Println("user accessed: " + requestedPath)

		if _, err := os.Stat(requestedPath); os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, requestedPath)
	}
}

func allowWebTypesOnly(log *log.Logger) func(h http.HandlerFunc) http.HandlerFunc {
	allowedExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".svg":  true,
		".webp": true,
		".ico":  true,
		".bmp":  true,
		".tiff": true,

		".js":  true,
		".mjs": true,

		".css": true,

		".woff":  true,
		".woff2": true,
		".ttf":   true,
		".eot":   true,

		".pdf": true,
		".txt": true,
	}

	return shared.RestrictExtensions(log, allowedExtensions)
}
