package redis

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// Enabled reports whether Redis is configured. Redis is optional: when
// REDIS_HOST is empty, features that use it fall back to in-memory storage.
func Enabled() bool {
	return os.Getenv("REDIS_HOST") != ""
}

// NewClient connects to Redis and returns an error when it cannot be reached,
// so callers can decide to continue without it.
func NewClient() (*redis.Client, error) {
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	password := os.Getenv("REDIS_PASSWORD")

	if port == "" {
		port = "6379"
	}

	client := redis.NewClient(&redis.Options{
		Addr:        fmt.Sprintf("%s:%s", host, port),
		Password:    password,
		DB:          0, // use default DB
		DialTimeout: 2 * time.Second,
		ReadTimeout: time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("could not connect to redis at %s:%s: %w", host, port, err)
	}

	return client, nil
}
