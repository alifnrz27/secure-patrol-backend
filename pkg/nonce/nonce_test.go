package nonce

import (
	"context"
	"errors"
	"testing"
	"time"
)

type failingStore struct{}

func (failingStore) Use(context.Context, string, time.Duration) (bool, error) {
	return false, errors.New("redis: connection refused")
}

func TestMemoryStoreRejectsReusedNonce(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	if fresh, _ := store.Use(ctx, "app:nonce-1", time.Minute); !fresh {
		t.Fatal("first use should be fresh")
	}
	if fresh, _ := store.Use(ctx, "app:nonce-1", time.Minute); fresh {
		t.Fatal("second use should be rejected")
	}
	if fresh, _ := store.Use(ctx, "app:nonce-2", time.Minute); !fresh {
		t.Fatal("a different nonce should be fresh")
	}
}

func TestMemoryStoreAllowsNonceAfterExpiry(t *testing.T) {
	store := newMemoryStore()
	ctx := context.Background()

	store.Use(ctx, "app:nonce", time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	if fresh, _ := store.Use(ctx, "app:nonce", time.Minute); !fresh {
		t.Fatal("expired nonce should be usable again")
	}
}

func TestFallbackStoreUsesMemoryWhenRedisFails(t *testing.T) {
	store := &fallbackStore{primary: failingStore{}, fallback: newMemoryStore()}
	ctx := context.Background()

	fresh, err := store.Use(ctx, "app:nonce", time.Minute)
	if err != nil || !fresh {
		t.Fatalf("expected fresh nonce without error, got fresh=%v err=%v", fresh, err)
	}

	// Replay protection must keep working through the fallback.
	fresh, err = store.Use(ctx, "app:nonce", time.Minute)
	if err != nil || fresh {
		t.Fatalf("expected replay to be rejected without error, got fresh=%v err=%v", fresh, err)
	}
}

func TestDefaultFallsBackToMemoryWhenRedisUnreachable(t *testing.T) {
	t.Setenv("REDIS_HOST", "127.0.0.1")
	t.Setenv("REDIS_PORT", "1") // nothing listens here

	started := time.Now()
	store := Default()

	if _, ok := store.(*memoryStore); !ok {
		t.Fatalf("expected in-memory store, got %T", store)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("startup took too long without redis: %v", elapsed)
	}
}
