package auth

import (
	"encoding/json"
	"net/http"
)

func RequireAuth(jwtSecret string, authEnabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authEnabled {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(CookieName)
			if err != nil {
				writeUnauthorized(w)
				return
			}

			if _, err := ValidateToken(jwtSecret, cookie.Value); err != nil {
				writeUnauthorized(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"message": "unauthorized"},
	})
}
