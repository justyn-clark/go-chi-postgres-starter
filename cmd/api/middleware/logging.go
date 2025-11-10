package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// RequestLogger returns a middleware that logs HTTP requests
func RequestLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Process request
			next.ServeHTTP(ww, r)

			// Log request
			logger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Int("status", ww.Status()).
				Int("bytes", ww.BytesWritten()).
				Dur("duration", time.Since(start)).
				Msg("http request")
		})
	}
}

// SimpleRequestLogger is a simpler version that uses the global logger
func SimpleRequestLogger(next http.Handler) http.Handler {
	return RequestLogger(log.Logger)(next)
}
