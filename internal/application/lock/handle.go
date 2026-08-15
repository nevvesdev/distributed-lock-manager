package lock

import (
	"context"
	"log/slog"
	"time"

	"github.com/nevvesdev/distributed-lock-manager/internal/domain/lock"
)

// LockHandle representa um lock ativo com renovação automática via heartbeat.
// O caller deve sempre chamar Release ao terminar, ou cancelar o contexto pai.
type LockHandle struct {
	Lock    *lock.Lock
	cancel  context.CancelFunc
	service *LockService
	done    chan struct{}
}

// Release libera o lock e para o heartbeat.
func (h *LockHandle) Release(ctx context.Context) error {
	h.cancel()
	<-h.done
	return h.service.Release(ctx, h.Lock.Key, h.Lock.Owner, h.Lock.FencingToken)
}

// heartbeat renova o TTL do lock em intervalos de ttl/3 até o contexto ser cancelado.
func (h *LockHandle) heartbeat(ctx context.Context, ttl time.Duration) {
	defer close(h.done)

	interval := ttl / 3
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := h.service.repo.Renew(ctx, h.Lock.Key, h.Lock.Owner, h.Lock.FencingToken, ttl)
			if err != nil {
				slog.Warn("heartbeat: falha ao renovar lock",
					"key", h.Lock.Key,
					"owner", h.Lock.Owner,
					"token", h.Lock.FencingToken,
					"erro", err,
				)
				return
			}
			slog.Debug("heartbeat: lock renovado",
				"key", h.Lock.Key,
				"owner", h.Lock.Owner,
				"token", h.Lock.FencingToken,
				"proximo_em", interval,
			)
		}
	}
}
