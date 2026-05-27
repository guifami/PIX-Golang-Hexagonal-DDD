package httpadapter

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// SecurityHeaders sets standard HTTP security response headers.
// Swagger routes receive a permissive CSP; all other routes use the strict default-src 'none'.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Cache-Control", "no-store")

		if strings.HasPrefix(c.Request.URL.Path, "/swagger/") {
			c.Header("Content-Security-Policy",
				"default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'")
		} else {
			c.Header("Content-Security-Policy", "default-src 'none'")
		}
		c.Next()
	}
}

// CORS handles Cross-Origin Resource Sharing with a configurable allowlist.
// Pass "*" to allow all origins (development only).
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(allowedOrigins))
	allowAll := false
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
			break
		}
		allowed[strings.ToLower(o)] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			c.Next()
			return
		}

		if allowAll || allowed[strings.ToLower(origin)] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-API-Key,X-Idempotency-Key")
			c.Header("Access-Control-Max-Age", "86400")
			c.Header("Vary", "Origin")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// ipLimiter holds a rate.Limiter per IP and a last-seen timestamp for cleanup.
type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter enforces a per-IP token-bucket rate limit.
type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*ipLimiter
	rps     rate.Limit
	burst   int
}

func NewRateLimiter(rps int, burst int) *RateLimiter {
	rl := &RateLimiter{
		clients: make(map[string]*ipLimiter),
		rps:     rate.Limit(rps),
		burst:   burst,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) get(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	entry, ok := rl.clients[ip]
	if !ok {
		entry = &ipLimiter{limiter: rate.NewLimiter(rl.rps, rl.burst)}
		rl.clients[ip] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, entry := range rl.clients {
			if time.Since(entry.lastSeen) > 10*time.Minute {
				delete(rl.clients, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !rl.get(ip).Allow() {
			log := LoggerFromContext(c)
			log.Warn("rate limit exceeded", zap.String("ip", ip))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, ErrorResponse{Message: "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

// APIKeyAuth validates the X-API-Key header against a list of valid keys.
// Health and Swagger endpoints are exempt.
func APIKeyAuth(validKeys []string) gin.HandlerFunc {
	keySet := make(map[string]bool, len(validKeys))
	for _, k := range validKeys {
		if k != "" {
			keySet[k] = true
		}
	}

	return func(c *gin.Context) {
		if len(keySet) == 0 {
			// Auth disabled — no keys configured.
			c.Next()
			return
		}

		key := c.GetHeader("X-API-Key")
		if !keySet[key] {
			log := LoggerFromContext(c)
			log.Warn("unauthorized request", zap.String("ip", c.ClientIP()))
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Message: "invalid or missing API key"})
			return
		}
		c.Next()
	}
}

// RequestTimeout aborts the handler if it doesn't finish within the given duration.
func RequestTimeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(d):
			log := LoggerFromContext(c)
			log.Warn("request timeout", zap.String("path", c.Request.URL.Path))
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, ErrorResponse{Message: "request timeout"})
		}
	}
}

// recoverWithLogger is a panic-recovery handler that logs the panic with zap.
func recoverWithLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log := LoggerFromContext(c)
				log.Error("panic recovered", zap.Any("error", r))
				c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
			}
		}()
		c.Next()
	}
}

// ParseAPIKeys splits a comma-separated API key string into a slice.
func ParseAPIKeys(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	keys := make([]string, 0, len(parts))
	for _, p := range parts {
		if k := strings.TrimSpace(p); k != "" {
			keys = append(keys, k)
		}
	}
	return keys
}

// ParseAllowedOrigins splits a comma-separated origins string into a slice.
func ParseAllowedOrigins(raw string) []string {
	if raw == "" {
		return []string{"*"}
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if o := strings.TrimSpace(p); o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}
