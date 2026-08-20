package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/martin3zra/playsql"
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
}

// UpdatedAt is stamped automatically by playsql's Upsert (any column named
// exactly "updated_at" is set to time.Now() on every call) — that's the
// desired behavior here, since Set should always refresh it.
type cacheModel struct {
	Key       string    `db:"key"`
	Payload   []byte    `db:"payload"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (cacheModel) TableName() string { return "preview_cache" }

// SQLCache is a Cache backed by the preview_cache table, portable across every
// dialect playsql supports.
type SQLCache struct {
	db *playsql.DB
}

func NewSQLCache(db *playsql.DB) *SQLCache {
	return &SQLCache{db: db}
}

func (c *SQLCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	var row cacheModel
	err := c.db.Model(&cacheModel{}).WhereEq("key", key).First(ctx, &row)
	if errors.Is(err, playsql.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	return row.Payload, true, nil
}

func (c *SQLCache) Set(ctx context.Context, key string, value []byte) error {
	_, err := c.db.Model(&cacheModel{}).Upsert(
		ctx,
		[]map[string]any{{
			"key":     key,
			"payload": value,
		}},
		[]string{"key"},
		[]string{"payload", "updated_at"},
	)
	return err
}

func (c *SQLCache) Delete(ctx context.Context, key string) error {
	_, err := c.db.Model(&cacheModel{}).WhereEq("key", key).Delete(ctx)
	return err
}

func Remember[T any](
	ctx context.Context,
	c Cache,
	key string,
	fn func() (T, error),
) (T, error) {

	var zero T

	if data, ok, err := c.Get(ctx, key); err != nil {
		return zero, err
	} else if ok {

		var v T
		if err := json.Unmarshal(data, &v); err == nil {
			return v, nil
		}
		_ = c.Delete(ctx, key)
	}

	v, err := fn()
	if err != nil {
		return zero, err
	}

	if data, err := json.Marshal(v); err == nil {
		_ = c.Set(ctx, key, data)
	}

	return v, nil
}
