package pages

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/pkg/db"
	"github.com/google/uuid"
)

type Cursor string

type Paginator[T any] struct {
	Identifier  Identifier[T]
	db          db.KVDBer
	bitset      *Bitset
	initialized bool
}

func NewPaginator[T any](id Identifier[T], db db.KVDBer) *Paginator[T] {
	return &Paginator[T]{
		Identifier:  id,
		db:          db,
		bitset:      NewBitset(),
		initialized: false,
	}
}

type Identifier[T any] interface {
	GetPrefix() string
	GetID(T) int32
}

func GenCursor() Cursor {
	return Cursor(uuid.NewString())
}

func validPrefix(prefix string) bool {
	return !strings.Contains(prefix, ":")
}

func createPrefix[T any](id Identifier[T], c Cursor) (string, error) {
	prefix := id.GetPrefix()

	if !validPrefix(prefix) {
		return "", fmt.Errorf("invalid prefix `%s` given by %T", prefix, id)
	}

	return fmt.Sprintf("pagination:%#v:%#v:", prefix, c), nil
}

func (p *Paginator[T]) Init(ctx context.Context, c Cursor) error {
	prefix, err := createPrefix[T](p.Identifier, c)
	if err != nil {
		return err
	}

	// list, cursor, err := p.db.List(ctx, &prefix, nil)
	list, _, err := p.db.List(ctx, &prefix, nil)
	if err != nil {
		return err
	}

	kv, err := p.db.Get(ctx, list...)
	if err != nil {
		return nil
	}
	for k, v := range kv {
		power, err := strconv.ParseUint(strings.Split(k, ":")[3], 10, 0)
		if err != nil {
			continue
		}
		used, err := strconv.ParseUint(string(v), 10, 0)
		if err != nil {
			continue
		}

		p.bitset.Bits[power] = used
	}

	p.initialized = true

	return nil
}

func (Paginator[T]) Add(i T) {

}

func (p *Paginator[T]) Close() error {
	return p.db.Close()
}
