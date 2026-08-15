package redis

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"time"

	"github.com/nevvesdev/distributed-lock-manager/internal/domain/lock"
	"github.com/nevvesdev/distributed-lock-manager/pkg/fencing"
	"github.com/redis/go-redis/v9"
)

//go:embed scripts/acquire.lua
var acquireScript string

//go:embed scripts/release.lua
var releaseScript string

//go:embed scripts/renew.lua
var renewScript string

type LockRepository struct {
	client  *redis.Client
	acquire *redis.Script
	release *redis.Script
	renew   *redis.Script
}

func NewLockRepository(client *redis.Client) *LockRepository {
	return &LockRepository{
		client:  client,
		acquire: redis.NewScript(acquireScript),
		release: redis.NewScript(releaseScript),
		renew:   redis.NewScript(renewScript),
	}
}

func (r *LockRepository) Acquire(ctx context.Context, key, owner string, ttl time.Duration) (*lock.Lock, error) {
	lockKey := fencing.LockKey(key)
	tokenKey := fencing.TokenKey(key)
	ttlMs := ttl.Milliseconds()

	result, err := r.acquire.Run(ctx, r.client,
		[]string{lockKey, tokenKey},
		owner, ttlMs,
	).Int64()
	if err != nil {
		return nil, fmt.Errorf("erro ao executar acquire no Redis: %w", err)
	}

	if result == -1 {
		return nil, lock.ErrLockAlreadyHeld
	}

	now := time.Now()
	return &lock.Lock{
		Key:          key,
		Owner:        owner,
		FencingToken: result,
		TTL:          ttl,
		AcquiredAt:   now,
		ExpiresAt:    now.Add(ttl),
	}, nil
}

func (r *LockRepository) Release(ctx context.Context, key, owner string, fencingToken int64) error {
	lockKey := fencing.LockKey(key)
	tokenKey := fencing.TokenKey(key)

	result, err := r.release.Run(ctx, r.client,
		[]string{lockKey, tokenKey},
		owner, fencingToken,
	).Int64()
	if err != nil {
		return fmt.Errorf("erro ao executar release no Redis: %w", err)
	}

	switch result {
	case 1:
		return nil
	case 0:
		return lock.ErrLockNotFound
	case -1:
		return lock.ErrLockNotOwned
	case -2:
		return lock.ErrFencingTokenMismatch
	default:
		return fmt.Errorf("resultado inesperado do script release: %d", result)
	}
}

func (r *LockRepository) Renew(ctx context.Context, key, owner string, fencingToken int64, ttl time.Duration) error {
	lockKey := fencing.LockKey(key)
	tokenKey := fencing.TokenKey(key)
	ttlMs := ttl.Milliseconds()

	result, err := r.renew.Run(ctx, r.client,
		[]string{lockKey, tokenKey},
		owner, fencingToken, ttlMs,
	).Int64()
	if err != nil {
		return fmt.Errorf("erro ao executar renew no Redis: %w", err)
	}

	switch result {
	case 1:
		return nil
	case 0:
		return lock.ErrLockNotFound
	case -1:
		return lock.ErrLockNotOwned
	case -2:
		return lock.ErrFencingTokenMismatch
	default:
		return fmt.Errorf("resultado inesperado do script renew: %d", result)
	}
}

func (r *LockRepository) Get(ctx context.Context, key string) (*lock.Lock, error) {
	lockKey := fencing.LockKey(key)
	tokenKey := fencing.TokenKey(key)

	pipe := r.client.Pipeline()
	ownerCmd := pipe.Get(ctx, lockKey)
	tokenCmd := pipe.Get(ctx, tokenKey)
	ttlCmd := pipe.PTTL(ctx, lockKey)

	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, fmt.Errorf("erro ao buscar lock no Redis: %w", err)
	}

	owner, err := ownerCmd.Result()
	if err == redis.Nil {
		return nil, lock.ErrLockNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao ler owner do lock: %w", err)
	}

	tokenStr, err := tokenCmd.Result()
	if err != nil {
		return nil, fmt.Errorf("erro ao ler fencing token: %w", err)
	}

	token, err := strconv.ParseInt(tokenStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("fencing token invalido no Redis: %w", err)
	}

	remaining, err := ttlCmd.Result()
	if err != nil {
		return nil, fmt.Errorf("erro ao ler TTL do lock: %w", err)
	}

	now := time.Now()
	return &lock.Lock{
		Key:          key,
		Owner:        owner,
		FencingToken: token,
		TTL:          remaining,
		AcquiredAt:   now.Add(-remaining),
		ExpiresAt:    now.Add(remaining),
	}, nil
}
