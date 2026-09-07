package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// NewRouter configures the HTTP transport layer. Business modules can register
// their routes here as they are introduced.
func NewRouter(logger *slog.Logger, allowedOrigins []string) http.Handler {
	router := chi.NewRouter()
	router.Use(requestLogger(logger))
	router.Use(cors(allowedOrigins))

	router.Get("/api/health", health)
	router.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "The requested resource was not found.")
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "The request method is not allowed for this resource.")
	})

	return router
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		// The response status has already been written. Handlers should only pass
		// JSON-serializable values to this helper.
		return
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func cors(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			}

			if r.Method == http.MethodOptions {
				if origin != "" {
					if _, ok := allowed[origin]; !ok {
						writeError(w, http.StatusForbidden, "CORS_ORIGIN_NOT_ALLOWED", "Origin is not allowed.")
						return
					}
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			next.ServeHTTP(w, r)
			logger.Debug("HTTP request completed", "method", r.Method, "path", r.URL.Path, "duration", time.Since(started).String())
		})
	}
}
