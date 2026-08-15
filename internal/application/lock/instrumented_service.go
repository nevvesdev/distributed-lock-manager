package lock

import (
	"context"
	"errors"
	"time"

	domlock "github.com/nevvesdev/distributed-lock-manager/internal/domain/lock"
	"github.com/nevvesdev/distributed-lock-manager/internal/infra/metrics"
)

// InstrumentedLockService envolve LockService adicionando métricas Prometheus.
type InstrumentedLockService struct {
	inner   *LockService
	metrics *metrics.Metrics
}

// NewInstrumentedLockService cria um serviço instrumentado com métricas.
func NewInstrumentedLockService(inner *LockService, m *metrics.Metrics) *InstrumentedLockService {
	return &InstrumentedLockService{inner: inner, metrics: m}
}

func (s *InstrumentedLockService) Acquire(ctx context.Context, key, owner string, ttl time.Duration) (*domlock.Lock, error) {
	l, err := s.inner.Acquire(ctx, key, owner, ttl)
	if err != nil {
		if errors.Is(err, domlock.ErrLockAlreadyHeld) {
			s.metrics.AcquireTotal.WithLabelValues("failed").Inc()
			s.metrics.ContentionTotal.Inc()
		} else {
			s.metrics.AcquireTotal.WithLabelValues("error").Inc()
		}
		return nil, err
	}
	s.metrics.AcquireTotal.WithLabelValues("success").Inc()
	return l, nil
}

func (s *InstrumentedLockService) Release(ctx context.Context, key, owner string, fencingToken int64) error {
	return s.ReleaseWithDuration(ctx, key, owner, fencingToken, time.Time{})
}

// ReleaseWithDuration libera o lock e registra a duração se acquiredAt for fornecido.
func (s *InstrumentedLockService) ReleaseWithDuration(ctx context.Context, key, owner string, fencingToken int64, acquiredAt time.Time) error {
	err := s.inner.Release(ctx, key, owner, fencingToken)
	if err != nil {
		switch {
		case errors.Is(err, domlock.ErrLockNotFound):
			s.metrics.ReleaseTotal.WithLabelValues("not_found").Inc()
		case errors.Is(err, domlock.ErrLockNotOwned):
			s.metrics.ReleaseTotal.WithLabelValues("not_owned").Inc()
		case errors.Is(err, domlock.ErrFencingTokenMismatch):
			s.metrics.ReleaseTotal.WithLabelValues("token_mismatch").Inc()
		default:
			s.metrics.ReleaseTotal.WithLabelValues("error").Inc()
		}
		return err
	}
	s.metrics.ReleaseTotal.WithLabelValues("success").Inc()
	if !acquiredAt.IsZero() {
		s.metrics.HoldDuration.Observe(time.Since(acquiredAt).Seconds())
	}
	return nil
}

func (s *InstrumentedLockService) Renew(ctx context.Context, key, owner string, fencingToken int64, ttl time.Duration) error {
	err := s.inner.Renew(ctx, key, owner, fencingToken, ttl)
	if err != nil {
		switch {
		case errors.Is(err, domlock.ErrLockNotFound):
			s.metrics.RenewTotal.WithLabelValues("not_found").Inc()
		case errors.Is(err, domlock.ErrLockNotOwned):
			s.metrics.RenewTotal.WithLabelValues("not_owned").Inc()
		case errors.Is(err, domlock.ErrFencingTokenMismatch):
			s.metrics.RenewTotal.WithLabelValues("token_mismatch").Inc()
		default:
			s.metrics.RenewTotal.WithLabelValues("error").Inc()
		}
		return err
	}
	s.metrics.RenewTotal.WithLabelValues("success").Inc()
	return nil
}

func (s *InstrumentedLockService) Get(ctx context.Context, key string) (*domlock.Lock, error) {
	return s.inner.Get(ctx, key)
}

func (s *InstrumentedLockService) AcquireWithHeartbeat(ctx context.Context, key, owner string, ttl time.Duration) (*LockHandle, error) {
	return s.inner.AcquireWithHeartbeat(ctx, key, owner, ttl)
}
