package lock

import (
	"context"
	"time"

	domlock "github.com/nevvesdev/distributed-lock-manager/internal/domain/lock"
)

// Service define o contrato do serviço de lock usado pelos handlers HTTP.
type Service interface {
	Acquire(ctx context.Context, key, owner string, ttl time.Duration) (*domlock.Lock, error)
	Release(ctx context.Context, key, owner string, fencingToken int64) error
	Renew(ctx context.Context, key, owner string, fencingToken int64, ttl time.Duration) error
	Get(ctx context.Context, key string) (*domlock.Lock, error)
	AcquireWithHeartbeat(ctx context.Context, key, owner string, ttl time.Duration) (*LockHandle, error)
}
