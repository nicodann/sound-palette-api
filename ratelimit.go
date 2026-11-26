package main

import (
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	minute   *rate.Limiter
	day      *rate.Limiter
	lastSeen time.Time
}

type rateLimiter struct {
	mu        sync.Mutex
	visitors  map[string]*visitor
	global    *rate.Limiter
	perMinute int
	perDay    int
}

func newRateLimiter(perMinute, perDay, globalPerDay int) *rateLimiter {
	if perMinute <= 0 || perDay <= 0 { panic("rate limits must be positive") }
	
	limiter := &rateLimiter{
		visitors:  make(map[string]*visitor),
		perMinute: perMinute,
		perDay:    perDay,
	}
	if globalPerDay > 0 {
		limiter.global = rate.NewLimiter(rate.Every(24*time.Hour/time.Duration(globalPerDay)), globalPerDay)
	}
	go limiter.cleanup()
	return limiter
}

func (limiter *rateLimiter) getVisitor(ip string) *visitor {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	v, ok := limiter.visitors[ip]
	if !ok {
		v = &visitor{
			minute: rate.NewLimiter(rate.Every(time.Minute/time.Duration(limiter.perMinute)), limiter.perMinute),
			day:    rate.NewLimiter(rate.Every(24*time.Hour/time.Duration(limiter.perDay)), limiter.perDay),
		}
		limiter.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	return v
}

// Forget IPs idle for a day, so the map doesn't grow forever.
// After 24h idle their buckets would be full again anyway.
func (limiter *rateLimiter) cleanup() {
	for {
		time.Sleep(10 * time.Minute)
		limiter.mu.Lock()
		for ip, v := range limiter.visitors {
			if time.Since(v.lastSeen) > 24*time.Hour {
				delete(limiter.visitors, ip)
			}
		}
		limiter.mu.Unlock()
	}
}

func (limiter *rateLimiter) middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v := limiter.getVisitor(clientIP(r))

		if !v.minute.Allow() || !v.day.Allow() {
			http.Error(w, "Too many requests, please slow down.", http.StatusTooManyRequests)
			return
		}
		if limiter.global != nil && !limiter.global.Allow() {
			http.Error(w, "The demo has reached today's limit. Try again tomorrow.", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

func clientIP(r *http.Request) string {
	// Only trust this header when you're behind your host's proxy.
	// Locally, anyone could send a fake one
	if os.Getenv("TRUST_PROXY") == "true" {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
