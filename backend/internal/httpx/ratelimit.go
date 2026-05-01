package httpx

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// TokenBucket is a tiny in-memory rate limiter keyed by string (e.g.
// client IP). For multi-instance deployments swap for a Redis-backed
// implementation; for thesis-MVP single-process this is sufficient.
//
// Stale entries are pruned opportunistically every Allow() call (1 in N
// chance) so the state map cannot grow unbounded under sustained traffic
// from many distinct IPs. A bucket is considered stale when its
// lastRefill is older than `staleAfter` (default: 1 hour) AND it would
// have been refilled to capacity by now (so removing it has no effect on
// the limiter's behavior).
type TokenBucket struct {
	capacity   int
	refill     time.Duration
	staleAfter time.Duration
	mu         sync.Mutex
	state      map[string]*bucketState
	calls      uint64 // for opportunistic prune sampling
}

type bucketState struct {
	tokens     float64
	lastRefill time.Time
}

// NewTokenBucket creates a per-key bucket of `capacity` requests that
// refills one token every `refill` duration. e.g. capacity=5, refill=12s
// allows ~5 attempts per minute per key. Stale buckets are pruned after
// 1 hour of inactivity.
func NewTokenBucket(capacity int, refill time.Duration) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		refill:     refill,
		staleAfter: time.Hour,
		state:      make(map[string]*bucketState),
	}
}

// pruneInterval — every N Allow() calls we sweep stale entries. Picked
// to amortize sweeping across many requests rather than block any one.
const pruneInterval = 256

// Allow returns true and consumes a token if one is available; false if
// the caller should be rate-limited.
func (b *TokenBucket) Allow(key string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	st, ok := b.state[key]
	if !ok {
		st = &bucketState{tokens: float64(b.capacity), lastRefill: now}
		b.state[key] = st
	}
	elapsed := now.Sub(st.lastRefill)
	add := elapsed.Seconds() / b.refill.Seconds()
	st.tokens += add
	if st.tokens > float64(b.capacity) {
		st.tokens = float64(b.capacity)
	}
	st.lastRefill = now

	b.calls++
	if b.calls%pruneInterval == 0 {
		b.pruneLocked(now)
	}

	if st.tokens >= 1 {
		st.tokens--
		return true
	}
	return false
}

// pruneLocked removes buckets whose lastRefill is older than staleAfter.
// Caller must hold b.mu. A stale bucket has been idle long enough to
// fully refill; removing it just frees memory — the next Allow() recreates
// a fresh full bucket which is identical to what the caller would have
// had if we left the entry.
func (b *TokenBucket) pruneLocked(now time.Time) {
	cutoff := now.Add(-b.staleAfter)
	for k, st := range b.state {
		if st.lastRefill.Before(cutoff) {
			delete(b.state, k)
		}
	}
}

// Size returns the number of tracked buckets. Useful for tests and
// metrics; not part of the limiting hot path.
func (b *TokenBucket) Size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.state)
}

// RateLimitMiddleware returns a gin middleware that rate-limits per
// client IP. On 429 it sets a Retry-After header in seconds.
func RateLimitMiddleware(b *TokenBucket) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := clientIP(c)
		if !b.Allow(ip) {
			retry := int(b.refill.Seconds())
			if retry < 1 {
				retry = 1
			}
			c.Header("Retry-After", itoa(retry))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, ErrorResponse{
				Error: "too many requests, slow down",
			})
			return
		}
		c.Next()
	}
}

// clientIP returns the most-trustworthy client IP available. Behind a
// reverse proxy you'd want to honour X-Forwarded-For — for the thesis
// MVP we use the connection-level remote IP.
func clientIP(c *gin.Context) string {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}

// itoa is a small allocation-free integer-to-decimal-string helper used
// in Retry-After headers; avoids pulling in strconv for one call.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
