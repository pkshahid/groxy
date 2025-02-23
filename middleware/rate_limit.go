package middleware

import (
	"net/http"

	"golang.org/x/time/rate"
)

var limiter = rate.NewLimiter(1, 5) // 1 request per second, burst of 5

// RateLimitMiddleware limits requests
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
