package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type window struct {
	count   int
	resetAt time.Time
}

type rateLimiter struct {
	mu     sync.Mutex
	hits   map[string]*window
	limit  int
	period time.Duration
	lastGC time.Time
}

// allow menghitung hit per key dengan fixed window; state hanya di memori satu instance.
func (r *rateLimiter) allow(key string, now time.Time) (bool, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Bersihkan window kedaluwarsa agar map tidak tumbuh tanpa batas.
	if now.Sub(r.lastGC) > r.period {
		for k, w := range r.hits {
			if now.After(w.resetAt) {
				delete(r.hits, k)
			}
		}
		r.lastGC = now
	}

	w, ok := r.hits[key]
	if !ok || now.After(w.resetAt) {
		r.hits[key] = &window{count: 1, resetAt: now.Add(r.period)}
		return true, 0
	}

	if w.count >= r.limit {
		return false, w.resetAt.Sub(now)
	}

	w.count++
	return true, 0
}

// RateLimit membatasi request per IP klien untuk route yang dipasangi.
func RateLimit(limit int, period time.Duration) gin.HandlerFunc {
	limiter := &rateLimiter{
		hits:   make(map[string]*window),
		limit:  limit,
		period: period,
	}

	return func(c *gin.Context) {
		allowed, retryAfter := limiter.allow(c.ClientIP(), time.Now())
		if !allowed {
			seconds := int(retryAfter.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"message": "Too many requests, please try again later",
			})
			return
		}

		c.Next()
	}
}
