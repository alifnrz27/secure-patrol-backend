package nonce

import (
	"context"
	stdlog "log"
	"sync"
	"time"

	"secure-patrol-backend/pkg/log"
	appredis "secure-patrol-backend/pkg/redis"

	"github.com/redis/go-redis/v9"
)

// Store remembers used request nonces to block replayed requests.
type Store interface {
	// Use marks the nonce as used. It returns false when the nonce was already used.
	Use(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

var (
	defaultStore Store
	once         sync.Once
)

// Default returns the nonce store. Redis is optional:
//   - REDIS_HOST empty           -> in-memory store
//   - Redis unreachable at start -> in-memory store (with a warning)
//   - Redis fails while running  -> falls back to memory until Redis is back
//
// The in-memory store only protects against replays within one instance, so
// use Redis when running more than one instance.
func Default() Store {
	once.Do(func() {
		memory := newMemoryStore()

		if !appredis.Enabled() {
			stdlog.Println("[nonce] REDIS_HOST is not set, using in-memory nonce store (single instance only)")
			defaultStore = memory
			return
		}

		client, err := appredis.NewClient()
		if err != nil {
			stdlog.Printf("[nonce] WARNING %v; continuing with in-memory nonce store (single instance only)", err)
			defaultStore = memory
			return
		}

		stdlog.Println("[nonce] using redis nonce store")
		defaultStore = &fallbackStore{
			primary:  &redisStore{client: client},
			fallback: memory,
		}
	})

	return defaultStore
}

type redisStore struct {
	client *redis.Client
}

func (s *redisStore) Use(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return s.client.SetNX(ctx, "secure-patrol:nonce:"+key, 1, ttl).Result()
}

// fallbackStore uses primary (Redis) and switches to fallback (memory) for any
// request where primary returns an error, so a Redis outage does not reject requests.
type fallbackStore struct {
	primary  Store
	fallback Store

	mu         sync.Mutex
	lastWarnAt time.Time
}

func (s *fallbackStore) Use(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	fresh, err := s.primary.Use(ctx, key, ttl)
	if err == nil {
		return fresh, nil
	}

	s.warn(err)
	return s.fallback.Use(ctx, key, ttl)
}

// warn logs at most once per minute to avoid flooding the log during an outage.
func (s *fallbackStore) warn(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if time.Since(s.lastWarnAt) < time.Minute {
		return
	}
	s.lastWarnAt = time.Now()
	log.Warnf("redis nonce store unavailable, using in-memory fallback: %v", err)
}

type memoryStore struct {
	mu        sync.Mutex
	items     map[string]time.Time
	lastPurge time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{items: map[string]time.Time{}}
}

func (s *memoryStore) Use(_ context.Context, key string, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if now.Sub(s.lastPurge) > time.Minute {
		for k, expiresAt := range s.items {
			if now.After(expiresAt) {
				delete(s.items, k)
			}
		}
		s.lastPurge = now
	}

	if expiresAt, ok := s.items[key]; ok && now.Before(expiresAt) {
		return false, nil
	}

	s.items[key] = now.Add(ttl)
	return true, nil
}
