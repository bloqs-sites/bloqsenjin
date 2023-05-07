package db

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type KeyDB struct {
	rdb *redis.Client
}

func NewKeyDB(ctx context.Context, opt *redis.Options) (*KeyDB, error) {
	dbh := &KeyDB{
		rdb: redis.NewClient(opt),
	}

    // we should
	_ = dbh.rdb.Ping(ctx)

	return dbh, nil
}

func (db *KeyDB) Get(ctx context.Context, key ...string) (map[string][]byte, error) {
	res := make(map[string][]byte, len(key))

	for _, i := range key {
		v, err := db.rdb.Get(ctx, i).Result()

		if err != redis.Nil {
			res[i] = nil
		} else if err != nil {
			return nil, err
		} else {
			res[i] = []byte(v)
		}
	}

	return res, nil
}

func (db *KeyDB) Put(ctx context.Context, entries map[string][]byte, ttl time.Duration) error {
	for k, v := range entries {
		if err := db.rdb.Set(ctx, string(k), v, ttl).Err(); err != nil {
			return err
		}
	}

	return nil
}

func (db *KeyDB) Delete(ctx context.Context, key ...string) error {
	for _, i := range key {
		if err := db.rdb.Del(ctx, i).Err(); err != nil {
			return err
		}
	}

	return nil
}

func (db *KeyDB) List(ctx context.Context, prefix *string, limit *uint) (keys []string, cursor uint64, err error) {
	var max int64 = 1000
	if limit != nil {
		max = int64(*limit)
	}

	if limit != nil {
		keys, _, err = db.rdb.Scan(ctx, cursor, fmt.Sprintf("%s*", *prefix), max).Result()
	} else {
		keys, _, err = db.rdb.Scan(ctx, cursor, "*", max).Result()
	}

	return
}

func (db *KeyDB) DeleteAll(ctx context.Context) error {
	return db.rdb.FlushDBAsync(ctx).Err()
}

func (db *KeyDB) Head(ctx context.Context, key ...string) (bool, error) {
	count, err := db.rdb.Exists(ctx, key...).Result()

	return count >= int64(len(key)), err
}
