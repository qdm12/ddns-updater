package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
)

// basicAuth returns a middleware requiring HTTP basic authentication.
// Credentials are hashed first so the comparison takes a constant time
// regardless of their length.
func basicAuth(username, password string) func(http.Handler) http.Handler {
	expectedUser := sha256.Sum256([]byte(username))
	expectedPassword := sha256.Sum256([]byte(password))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if ok {
				userHash := sha256.Sum256([]byte(user))
				passHash := sha256.Sum256([]byte(pass))
				// Both are always compared, no short-circuiting.
				userOK := subtle.ConstantTimeCompare(userHash[:], expectedUser[:])
				passOK := subtle.ConstantTimeCompare(passHash[:], expectedPassword[:])
				if userOK&passOK == 1 {
					next.ServeHTTP(w, r)
					return
				}
			}
			w.Header().Set("WWW-Authenticate", `Basic realm="DDNS Updater", charset="UTF-8"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		})
	}
}
