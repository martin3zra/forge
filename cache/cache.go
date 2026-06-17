package cache

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/martin3zra/forge/database"
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte) error
	Delete(ctx context.Context, key string) error
}

type PgCache struct {
	q database.Querier
}

func NewPgCache(q database.Querier) *PgCache {
	return &PgCache{q: q}
}

func (c *PgCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	var payload []byte

	err := c.q.QueryRowContext(ctx, `
        SELECT payload
        FROM preview_cache
        WHERE key = $1
    `, key).Scan(&payload)

	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	return payload, true, nil
}

func (c *PgCache) Set(ctx context.Context, key string, value []byte) error {
	_, err := c.q.ExecContext(ctx, `
        INSERT INTO preview_cache (key, payload)
        VALUES ($1, $2)
        ON CONFLICT (key)
        DO UPDATE SET payload = EXCLUDED.payload, updated_at = now()
    `, key, value)

	return err
}

func (c *PgCache) Delete(ctx context.Context, key string) error {
	_, err := c.q.ExecContext(ctx, `
        DELETE FROM preview_cache WHERE key = $1
    `, key)
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
