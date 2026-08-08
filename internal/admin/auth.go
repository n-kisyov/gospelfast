package admin

import (
	"crypto/subtle"
	"net/http"
	"os"
)

func AuthMiddleware(next http.Handler) http.Handler {
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "gospelfast"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pass, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="gospelfast admin"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
