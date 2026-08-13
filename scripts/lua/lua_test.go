package lua_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return mr, client
}

func scriptContent(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	require.NoError(t, err)
	return string(b)
}

func TestAcquire_Sucesso(t *testing.T) {
	_, client := setup(t)
	ctx := context.Background()

	script := redis.NewScript(scriptContent(t, "acquire.lua"))
	result, err := script.Run(ctx, client,
		[]string{"dlm:lock:pagamento-1", "dlm:token:pagamento-1"},
		"worker-A", int(time.Second*30/time.Millisecond),
	).Int64()

	require.NoError(t, err)
	assert.Equal(t, int64(1), result, "primeiro acquire deve retornar token=1")
}

func TestAcquire_LockJaOcupado(t *testing.T) {
	_, client := setup(t)
	ctx := context.Background()

	script := redis.NewScript(scriptContent(t, "acquire.lua"))
	keys := []string{"dlm:lock:pagamento-2", "dlm:token:pagamento-2"}
	ttl := int(time.Second * 30 / time.Millisecond)

	_, err := script.Run(ctx, client, keys, "worker-A", ttl).Int64()
	require.NoError(t, err)

	result, err := script.Run(ctx, client, keys, "worker-B", ttl).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(0), result, "segundo acquire deve retornar 0 (ocupado)")
}

func TestRelease_Sucesso(t *testing.T) {
	_, client := setup(t)
	ctx := context.Background()

	acquireScript := redis.NewScript(scriptContent(t, "acquire.lua"))
	releaseScript := redis.NewScript(scriptContent(t, "release.lua"))
	keys := []string{"dlm:lock:pagamento-3", "dlm:token:pagamento-3"}
	ttl := int(time.Second * 30 / time.Millisecond)

	token, err := acquireScript.Run(ctx, client, keys, "worker-A", ttl).Int64()
	require.NoError(t, err)

	result, err := releaseScript.Run(ctx, client, keys, "worker-A", token).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), result, "release com token correto deve retornar 1")
}

func TestRelease_FencingTokenMismatch(t *testing.T) {
	_, client := setup(t)
	ctx := context.Background()

	acquireScript := redis.NewScript(scriptContent(t, "acquire.lua"))
	releaseScript := redis.NewScript(scriptContent(t, "release.lua"))
	keys := []string{"dlm:lock:pagamento-4", "dlm:token:pagamento-4"}
	ttl := int(time.Second * 30 / time.Millisecond)

	_, err := acquireScript.Run(ctx, client, keys, "worker-A", ttl).Int64()
	require.NoError(t, err)

	result, err := releaseScript.Run(ctx, client, keys, "worker-A", int64(999)).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(-2), result, "token mismatch deve retornar -2")
}

func TestRenew_Sucesso(t *testing.T) {
	_, client := setup(t)
	ctx := context.Background()

	acquireScript := redis.NewScript(scriptContent(t, "acquire.lua"))
	renewScript := redis.NewScript(scriptContent(t, "renew.lua"))
	keys := []string{"dlm:lock:pagamento-5", "dlm:token:pagamento-5"}
	ttl := int(time.Second * 30 / time.Millisecond)

	token, err := acquireScript.Run(ctx, client, keys, "worker-A", ttl).Int64()
	require.NoError(t, err)

	result, err := renewScript.Run(ctx, client, keys, "worker-A", token, ttl).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), result, "renew com token correto deve retornar 1")
}

func TestRenew_LockExpirado(t *testing.T) {
	mr, client := setup(t)
	ctx := context.Background()

	acquireScript := redis.NewScript(scriptContent(t, "acquire.lua"))
	renewScript := redis.NewScript(scriptContent(t, "renew.lua"))
	keys := []string{"dlm:lock:pagamento-6", "dlm:token:pagamento-6"}
	ttl := int(100) // 100ms

	token, err := acquireScript.Run(ctx, client, keys, "worker-A", ttl).Int64()
	require.NoError(t, err)

	mr.FastForward(200 * time.Millisecond)

	result, err := renewScript.Run(ctx, client, keys, "worker-A", token, int(time.Second*30/time.Millisecond)).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(0), result, "renew de lock expirado deve retornar 0")
}
