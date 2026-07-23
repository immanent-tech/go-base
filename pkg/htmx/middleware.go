package htmx

import (
	"net/http"
)

// SetupHTMX middleware performs general setup for serving HTMX-powered content.
func SetupHTMX(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Add("Vary", HeaderRequest)
		res.Header().Add("Vary", HeaderHistoryRestoreRequest)
		next.ServeHTTP(res, req)
	})
}

// RequireHTMX middleware will only pass control to the next handler if the request is HTMX powered. If not, it will
// return 403: Forbidden response.
func RequireHTMX(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if !IsHTMX(req) {
			http.Error(res, "Not allowed", http.StatusForbidden)
			return
		}
		next.ServeHTTP(res, req)
	})
}
