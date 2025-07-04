package server

import (
	"embed"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

//go:embed all:static
var embedFS embed.FS

// webRouter creates a router that represents the routes under /web
func webRouter() (http.Handler, error) {
	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		const targetPath = "static/index.html"
		htmlContent, err := embedFS.ReadFile(targetPath)
		if err != nil {
			http.Error(w, "Internal Server Error: Could not load page.", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(htmlContent)
	})

	return r, nil
}
