package api

import (
	"io/fs"
	"net/http"
	"strings"
	"time"
)

// SetupRouter configures the HTTP routes, static web dashboard, and middleware.
func SetupRouter(handler *Handler, staticFS fs.FS) http.Handler {
	mux := http.NewServeMux()

	// REST API routes
	mux.HandleFunc("/api/v1/health", handler.HandleHealth)
	mux.HandleFunc("/api/v1/verify", handler.HandleVerify)
	mux.HandleFunc("/api/v1/extract", handler.HandleExtractClaims)

	// Static Web UI Dashboard
	if staticFS != nil {
		fileServer := http.FileServer(http.FS(staticFS))
		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.NotFound(w, r)
				return
			}
			fileServer.ServeHTTP(w, r)
		}))
	}

	return withCORS(withLogging(mux))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		_ = start // Keep request processing lean and silent
	})
}
