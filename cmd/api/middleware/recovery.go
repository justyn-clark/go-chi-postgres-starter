package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog/log"
)

// Recoverer recovers from panics and logs the error
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Error().
					Interface("error", err).
					Bytes("stack", debug.Stack()).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Msg("panic recovered")

				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
