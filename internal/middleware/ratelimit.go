package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/asepcepot6140-glitch/netcore/internal/utils"
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

func newRateLimiter(limit int, period time.Duration) *rateLimiter {
	return &rateLimiter{
		hits:   make(map[string]*window),
		limit:  limit,
		period: period,
	}
}

func (r *rateLimiter) reject(c *gin.Context, retryAfter time.Duration) {
	seconds := int(retryAfter.Seconds()) + 1
	c.Header("Retry-After", strconv.Itoa(seconds))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"message": "Too many requests, please try again later",
	})
}

// RateLimit membatasi request per IP klien untuk route yang dipasangi.
func RateLimit(limit int, period time.Duration) gin.HandlerFunc {
	limiter := newRateLimiter(limit, period)

	return func(c *gin.Context) {
		allowed, retryAfter := limiter.allow(c.ClientIP(), time.Now())
		if !allowed {
			limiter.reject(c, retryAfter)
			return
		}

		c.Next()
	}
}

const (
	MaxPhoneKeyBodyBytes = 4096
	maxPhoneKeyLength    = 30
)

// RateLimitByPhone membatasi request per nomor telepon di body JSON ({"phone": "..."}),
// sehingga serangan dari banyak IP tetap dibatasi per akun target.
// Pasang middleware yang sama di beberapa route untuk berbagi satu kuota.
// Body dibaca lalu dikembalikan; request tanpa phone valid diteruskan agar ditolak handler.
func RateLimitByPhone(limit int, period time.Duration) gin.HandlerFunc {
	limiter := newRateLimiter(limit, period)

	return func(c *gin.Context) {
		data, err := io.ReadAll(io.LimitReader(c.Request.Body, MaxPhoneKeyBodyBytes+1))
		if err != nil || len(data) > MaxPhoneKeyBodyBytes {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "Invalid request body"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(data))

		var payload struct {
			Phone string `json:"phone"`
		}
		if json.Unmarshal(data, &payload) != nil {
			c.Next()
			return
		}

		// Sama dengan normalisasi di service, supaya kunci cocok dengan akun yang dicari.
		phone := utils.NormalizePhone(payload.Phone)
		if phone == "" || len(payload.Phone) > maxPhoneKeyLength {
			c.Next()
			return
		}

		if allowed, retryAfter := limiter.allow(phone, time.Now()); !allowed {
			limiter.reject(c, retryAfter)
			return
		}

		c.Next()
	}
}
