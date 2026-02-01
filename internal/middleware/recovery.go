package middleware

import (
	"log/slog"
	"net/http"
)

// RecoveryMiddleware recovers from panics
type RecoveryMiddleware struct {
	logger *slog.Logger
}

// NewRecoveryMiddleware creates a new recovery middleware
func NewRecoveryMiddleware(logger *slog.Logger) *RecoveryMiddleware {
	return &RecoveryMiddleware{logger: logger}
}

// Wrap wraps an HTTP handler with panic recovery
func (m *RecoveryMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				m.logger.Error("panic recovered",
					"error", err,
					"path", r.URL.Path,
				)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"internal server error"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
