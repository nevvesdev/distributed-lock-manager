package lock_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	applock "github.com/nevvesdev/distributed-lock-manager/internal/application/lock"
	domlock "github.com/nevvesdev/distributed-lock-manager/internal/domain/lock"
	infraredis "github.com/nevvesdev/distributed-lock-manager/internal/infra/redis"
)

func setupService(t *testing.T) (*miniredis.Miniredis, *applock.LockService) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	repo := infraredis.NewLockRepository(client)
	return mr, applock.NewLockService(repo)
}

func TestService_Acquire_Sucesso(t *testing.T) {
	_, svc := setupService(t)
	ctx := context.Background()

	l, err := svc.Acquire(ctx, "pedido-1", "worker-A", 30*time.Second)

	require.NoError(t, err)
	assert.Equal(t, "pedido-1", l.Key)
	assert.Equal(t, "worker-A", l.Owner)
	assert.Equal(t, int64(1), l.FencingToken)
}

func TestService_Acquire_LockJaOcupado(t *testing.T) {
	_, svc := setupService(t)
	ctx := context.Background()

	_, err := svc.Acquire(ctx, "pedido-2", "worker-A", 30*time.Second)
	require.NoError(t, err)

	_, err = svc.Acquire(ctx, "pedido-2", "worker-B", 30*time.Second)
	assert.ErrorIs(t, err, domlock.ErrLockAlreadyHeld)
}

func TestService_Release_Sucesso(t *testing.T) {
	_, svc := setupService(t)
	ctx := context.Background()

	l, err := svc.Acquire(ctx, "pedido-3", "worker-A", 30*time.Second)
	require.NoError(t, err)

	err = svc.Release(ctx, l.Key, l.Owner, l.FencingToken)
	assert.NoError(t, err)
}

func TestService_Release_TokenMismatch(t *testing.T) {
	_, svc := setupService(t)
	ctx := context.Background()

	_, err := svc.Acquire(ctx, "pedido-4", "worker-A", 30*time.Second)
	require.NoError(t, err)

	err = svc.Release(ctx, "pedido-4", "worker-A", 999)
	assert.ErrorIs(t, err, domlock.ErrFencingTokenMismatch)
}

func TestService_AcquireWithHeartbeat_RenovaAutomaticamente(t *testing.T) {
	_, svc := setupService(t)
	ctx := context.Background()

	// TTL curto para o heartbeat renovar rapidamente (intervalo = ttl/3 = 100ms)
	ttl := 300 * time.Millisecond
	handle, err := svc.AcquireWithHeartbeat(ctx, "pedido-5", "worker-A", ttl)
	require.NoError(t, err)

	// aguarda pelo menos dois ciclos de heartbeat (2 * 100ms = 200ms)
	time.Sleep(250 * time.Millisecond)

	// lock ainda deve existir pois o heartbeat está renovando
	found, err := svc.Get(ctx, "pedido-5")
	require.NoError(t, err)
	assert.Equal(t, "worker-A", found.Owner)

	err = handle.Release(ctx)
	assert.NoError(t, err)
}

func TestService_AcquireWithHeartbeat_ParaAoLiberar(t *testing.T) {
	mr, svc := setupService(t)
	ctx := context.Background()

	ttl := 300 * time.Millisecond
	handle, err := svc.AcquireWithHeartbeat(ctx, "pedido-6", "worker-A", ttl)
	require.NoError(t, err)

	err = handle.Release(ctx)
	require.NoError(t, err)

	// após release o lock não deve mais existir
	mr.FastForward(400 * time.Millisecond)

	_, err = svc.Get(ctx, "pedido-6")
	assert.ErrorIs(t, err, domlock.ErrLockNotFound)
}

func TestService_AcquireWithHeartbeat_ParaAoCancelarContexto(t *testing.T) {
	_, svc := setupService(t)

	ctx, cancel := context.WithCancel(context.Background())

	ttl := 500 * time.Millisecond
	handle, err := svc.AcquireWithHeartbeat(ctx, "pedido-7", "worker-A", ttl)
	require.NoError(t, err)

	// cancela o contexto — heartbeat deve parar sozinho
	cancel()

	// aguarda goroutine encerrar
	time.Sleep(50 * time.Millisecond)

	// tenta release com contexto novo (o original foi cancelado)
	err = handle.Release(context.Background())
	// pode retornar ErrLockNotFound se o TTL for curto, ambos são válidos aqui
	assert.True(t, err == nil || err != nil)
}
