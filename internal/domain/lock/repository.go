package lock

import (
	"context"
	"time"
)

type LockRepository interface {
	Acquire(ctx context.Context, key, owner string, ttl time.Duration) (*Lock, error)
	Release(ctx context.Context, key, owner string, fencingToken int64) error
	Renew(ctx context.Context, key, owner string, fencingToken int64, ttl time.Duration) error
	Get(ctx context.Context, key string) (*Lock, error)
}
