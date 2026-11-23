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
	minute		*rate.Limiter
	day				*rate.Limiter
	lastSeen	time.Time
}

type aiRateLimiter struct {
	mu				sync.Mutex
	visitors	map[string]*visitor
	global 		*rate.Limiter
}

func newAIRateLimiter() *aiRateLimiter {
	l := &aiRateLimiter{
		visitors: make(map[string]*visitor),
		// 150 tokens, refilling one every 24h/150 (~ 9.6min)
		global: rate.NewLimiter(rate.Every(24*time.Hour/150), 150),
	}
	go l.cleanup()
	return l
}

func (l *aiRateLimiter) getVisitor(ip string) * visitor {
	l.mu.Lock()
	defer l.mu.Unlock()

	v, ok := l.visitors[ip]
	if !ok {
		v = &visitor{
			minute: rate.NewLimiter(rate.Every(time.Minute/5), 5),
			day:		rate.NewLimiter(rate.Every(24*time.Hour/30), 30),
		}
		l.visitors[ip] = v
	}
	v.lastSeen = time.Now()
	return v
}

// Forget IPs idle for a day, so the map doesn't grow forever.
// After 24h idle their buckets would be full again anyway.
func (l *aiRateLimiter) cleanup() {
	for {
		time.Sleep(10 * time.Minute)
		l.mu.Lock()
		for ip, v := range l.visitors {
			if time.Since(v.lastSeen) > 24*time.Hour {
				delete(l.visitors, ip)
			}
		}
		l.mu.Unlock()
	}
}

func (l *aiRateLimiter) middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r * http.Request) {
		v := l.getVisitor(clientIP(r))

		if !v.minute.Allow() || !v.day.Allow() {
			http.Error(w, "Too many requests, please slow down.", http.StatusTooManyRequests)
			return
		}
		if !l.global.Allow() {
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
			return strings.TrimSpace(parts[len(parts) -1])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}