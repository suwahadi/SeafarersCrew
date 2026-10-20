package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// HitCounter increments a fixed-window counter.
type HitCounter interface {
	Hit(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error)
}

// RateLimit allows limit requests per client IP within window for the named bucket.
func RateLimit(counter HitCounter, bucket string, limit int64, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			n, retry, err := counter.Hit(r.Context(), bucket+":"+ClientIP(r.Context()), window)
			if err != nil {
				resource.Error(w, r, err)
				return
			}
			if n > limit {
				w.Header().Set("Retry-After", strconv.Itoa(max(1, int(retry.Seconds()))))
				resource.Error(w, r, apperr.ErrTooManyAttempts)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
