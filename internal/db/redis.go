package db

import (
	"context"
	"fmt"
	"net"

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

func (db *KeyDB) Get(ctx context.Context, key []byte) error {
	return db.rdb.Get(ctx, string(key)).Err()
}

func (db *KeyDB) Put(ctx context.Context, key, value []byte) error {
	return db.rdb.Set(ctx, string(key), value, 0).Err()
}
