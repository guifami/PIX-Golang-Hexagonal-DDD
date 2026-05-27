package httpadapter

import (
	"bytes"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type idempotencyEntry struct {
	status    int
	body      []byte
	createdAt time.Time
}

// IdempotencyStore is an in-memory store for idempotency keys.
// A real production system would use Redis or a DB-backed store.
type IdempotencyStore struct {
	mu      sync.RWMutex
	entries map[string]*idempotencyEntry
	ttl     time.Duration
}

func NewIdempotencyStore(ttl time.Duration) *IdempotencyStore {
	s := &IdempotencyStore{
		entries: make(map[string]*idempotencyEntry),
		ttl:     ttl,
	}
	go s.cleanup()
	return s
}

func (s *IdempotencyStore) get(key string) (*idempotencyEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.entries[key]
	if !ok || time.Since(entry.createdAt) > s.ttl {
		return nil, false
	}
	return entry, true
}

func (s *IdempotencyStore) set(key string, status int, body []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = &idempotencyEntry{status: status, body: body, createdAt: time.Now()}
}

func (s *IdempotencyStore) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		for k, entry := range s.entries {
			if time.Since(entry.createdAt) > s.ttl {
				delete(s.entries, k)
			}
		}
		s.mu.Unlock()
	}
}

// responseRecorder captures the response body and status for caching.
type responseRecorder struct {
	gin.ResponseWriter
	buf    bytes.Buffer
	status int
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.buf.Write(b)
	return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// IdempotencyMiddleware ensures that POST requests with the same X-Idempotency-Key
// return the same response within the TTL window.
func IdempotencyMiddleware(store *IdempotencyStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}
		key := c.GetHeader("X-Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}

		if entry, ok := store.get(key); ok {
			c.Header("X-Idempotency-Replayed", "true")
			c.Data(entry.status, "application/json; charset=utf-8", entry.body)
			c.Abort()
			return
		}

		rec := &responseRecorder{ResponseWriter: c.Writer, status: http.StatusOK}
		c.Writer = rec
		c.Next()

		store.set(key, rec.status, rec.buf.Bytes())
	}
}
