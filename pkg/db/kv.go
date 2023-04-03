package db

import "context"

type KVDBer interface {
	Get(ctx context.Context, key []byte) error
	Put(ctx context.Context, key, value []byte) error
}
