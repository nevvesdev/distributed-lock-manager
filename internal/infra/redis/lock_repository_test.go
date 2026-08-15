package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nevvesdev/distributed-lock-manager/internal/domain/lock"
	infraredis "github.com/nevvesdev/distributed-lock-manager/internal/infra/redis"
)

func setupRepo(t *testing.T) (*miniredis.Miniredis, *infraredis.LockRepository) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return mr, infraredis.NewLockRepository(client)
}

func TestAcquire_Sucesso(t *testing.T) {
	_, repo := setupRepo(t)
	ctx := context.Background()

	l, err := repo.Acquire(ctx, "pedido-1", "worker-A", 30*time.Second)

	require.NoError(t, err)
	assert.Equal(t, "pedido-1", l.Key)
	assert.Equal(t, "worker-A", l.Owner)
	assert.Equal(t, int64(1), l.FencingToken)
	assert.False(t, l.IsExpired())
}

func TestAcquire_LockJaOcupado(t *testing.T) {
	_, repo := setupRepo(t)
	ctx := context.Background()

	_, err := repo.Acquire(ctx, "pedido-2", "worker-A", 30*time.Second)
	require.NoError(t, err)

	_, err = repo.Acquire(ctx, "pedido-2", "worker-B", 30*time.Second)
	assert.ErrorIs(t, err, lock.ErrLockAlreadyHeld)
}

func TestAcquire_TokenMonotonico(t *testing.T) {
	mr, repo := setupRepo(t)
	ctx := context.Background()

	l1, err := repo.Acquire(ctx, "pedido-3", "worker-A", 30*time.Second)
	require.NoError(t, err)
	assert.Equal(t, int64(1), l1.FencingToken)

	// libera e readquire para garantir incremento do token
	err = repo.Release(ctx, "pedido-3", "worker-A", l1.FencingToken)
	require.NoError(t, err)

	// avança tempo para garantir que o lock foi liberado
	mr.FastForward(time.Millisecond)

	l2, err := repo.Acquire(ctx, "pedido-3", "worker-B", 30*time.Second)
	require.NoError(t, err)
	assert.Equal(t, int64(2), l2.FencingToken, "token deve ser monotônico crescente")
}

func TestRelease_Sucesso(t *testing.T) {
	_, repo := setupRepo(t)
	ctx := context.Background()

	l, err := repo.Acquire(ctx, "pedido-4", "worker-A", 30*time.Second)
	require.NoError(t, err)

	err = repo.Release(ctx, "pedido-4", "worker-A", l.FencingToken)
	assert.NoError(t, err)
}

func TestRelease_FencingTokenMismatch(t *testing.T) {
	_, repo := setupRepo(t)
	ctx := context.Background()

	_, err := repo.Acquire(ctx, "pedido-5", "worker-A", 30*time.Second)
	require.NoError(t, err)

	err = repo.Release(ctx, "pedido-5", "worker-A", 999)
	assert.ErrorIs(t, err, lock.ErrFencingTokenMismatch)
}

func TestRelease_OwnerErrado(t *testing.T) {
	_, repo := setupRepo(t)
	ctx := context.Background()

	l, err := repo.Acquire(ctx, "pedido-6", "worker-A", 30*time.Second)
	require.NoError(t, err)

	err = repo.Release(ctx, "pedido-6", "worker-B", l.FencingToken)
	assert.ErrorIs(t, err, lock.ErrLockNotOwned)
}

func TestRenew_Sucesso(t *testing.T) {
	_, repo := setupRepo(t)
	ctx := context.Background()

	l, err := repo.Acquire(ctx, "pedido-7", "worker-A", 30*time.Second)
	require.NoError(t, err)

	err = repo.Renew(ctx, "pedido-7", "worker-A", l.FencingToken, 60*time.Second)
	assert.NoError(t, err)
}

func TestRenew_LockExpirado(t *testing.T) {
	mr, repo := setupRepo(t)
	ctx := context.Background()

	l, err := repo.Acquire(ctx, "pedido-8", "worker-A", 100*time.Millisecond)
	require.NoError(t, err)

	mr.FastForward(200 * time.Millisecond)

	err = repo.Renew(ctx, "pedido-8", "worker-A", l.FencingToken, 30*time.Second)
	assert.ErrorIs(t, err, lock.ErrLockNotFound)
}

func TestGet_Sucesso(t *testing.T) {
	_, repo := setupRepo(t)
	ctx := context.Background()

	l, err := repo.Acquire(ctx, "pedido-9", "worker-A", 30*time.Second)
	require.NoError(t, err)

	found, err := repo.Get(ctx, "pedido-9")
	require.NoError(t, err)
	assert.Equal(t, l.Key, found.Key)
	assert.Equal(t, l.Owner, found.Owner)
	assert.Equal(t, l.FencingToken, found.FencingToken)
}

func TestGet_NaoEncontrado(t *testing.T) {
	_, repo := setupRepo(t)
	ctx := context.Background()

	_, err := repo.Get(ctx, "chave-inexistente")
	assert.ErrorIs(t, err, lock.ErrLockNotFound)
}
