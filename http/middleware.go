package http

import "net/http"

// LimitRequestSize returns a middleware that limits the request body size to maxBytes.
// This prevents DoS attacks from large request bodies.
func LimitRequestSize(next http.Handler, maxBytes int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		}
		next.ServeHTTP(w, r)
	})
}
