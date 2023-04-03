package db

import (
	"context"
	"time"
)

type KVDBer interface {
	Get(ctx context.Context, key ...string) (map[string][]byte, error)
	Put(ctx context.Context, entries map[string][]byte, ttl time.Duration) error
	Delete(ctx context.Context, key ...string) error
    List(ctx context.Context, prefix *string, limit *uint) ([]string, uint64, error)
    DeleteAll(ctx context.Context) error
}
