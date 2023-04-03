package db

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/redis/go-redis/v9"
)

type KeyDB struct {
	rdb *redis.Client
}

type RedisCreds struct {
	domain string
	port   uint16
	pass   string
	db     int
}

func NewKeyDB(creds RedisCreds) *KeyDB {
	return &KeyDB{
		rdb: redis.NewClient(&redis.Options{
			Addr:     net.JoinHostPort(creds.domain, fmt.Sprint(creds.port)),
			Password: creds.pass,
			DB:       creds.db,
		}),
	}
}

func NewRedisCreds(domain string, port uint16, pass string, db int) RedisCreds {
	return RedisCreds{
		domain: domain,
		port:   port,
		pass:   pass,
		db:     db,
	}
}

func (db *KeyDB) Get(ctx context.Context, key ...string) (map[string][]byte, error) {
	res := make(map[string][]byte, len(key))

	var err error
	for _, i := range key {
		e := db.rdb.Get(ctx, i)

		if err = e.Err(); err != nil {
			return nil, err
		}

		if res[i], err = e.Bytes(); err != nil {
			return nil, err
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
