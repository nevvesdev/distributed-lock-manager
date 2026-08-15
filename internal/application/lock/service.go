package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/nevvesdev/distributed-lock-manager/internal/domain/lock"
)

// LockService orquestra operações de lock distribuído com suporte a heartbeat automático.
type LockService struct {
	repo lock.LockRepository
}

// cria um novo LockService com o repositório fornecido.
func NewLockService(repo lock.LockRepository) *LockService {
	return &LockService{repo: repo}
}

// tenta adquirir o lock para o owner com o TTL especificado.
func (s *LockService) Acquire(ctx context.Context, key, owner string, ttl time.Duration) (*lock.Lock, error) {
	l, err := s.repo.Acquire(ctx, key, owner, ttl)
	if err != nil {
		return nil, fmt.Errorf("service.Acquire: %w", err)
	}
	return l, nil
}

// libera o lock validando owner e fencing token.
func (s *LockService) Release(ctx context.Context, key, owner string, fencingToken int64) error {
	if err := s.repo.Release(ctx, key, owner, fencingToken); err != nil {
		return fmt.Errorf("service.Release: %w", err)
	}
	return nil
}

// renova o TTL do lock validando owner e fencing token.
func (s *LockService) Renew(ctx context.Context, key, owner string, fencingToken int64, ttl time.Duration) error {
	if err := s.repo.Renew(ctx, key, owner, fencingToken, ttl); err != nil {
		return fmt.Errorf("service.Renew: %w", err)
	}
	return nil
}

// Get retorna o estado atual do lock.
func (s *LockService) Get(ctx context.Context, key string) (*lock.Lock, error) {
	l, err := s.repo.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("service.Get: %w", err)
	}
	return l, nil
}

// adquire o lock e inicia uma goroutine que renova o TTL
// automaticamente em intervalos de ttl/3, até o contexto ser cancelado.
// retorna um LockHandle que deve ser usado para liberar o lock ao final.
func (s *LockService) AcquireWithHeartbeat(ctx context.Context, key, owner string, ttl time.Duration) (*LockHandle, error) {
	l, err := s.repo.Acquire(ctx, key, owner, ttl)
	if err != nil {
		return nil, fmt.Errorf("service.AcquireWithHeartbeat: %w", err)
	}

	handleCtx, cancel := context.WithCancel(ctx)

	handle := &LockHandle{
		Lock:    l,
		cancel:  cancel,
		service: s,
		done:    make(chan struct{}),
	}

	go handle.heartbeat(handleCtx, ttl)

	return handle, nil
}
