package helpers

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog"
)

// Recoverer (app)
// Catch all errors not handled anywhere else
func Recoverer(log *zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Msg(fmt.Sprintf("%+v", r))
					w.WriteHeader(500)
					return
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
