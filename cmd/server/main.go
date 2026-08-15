package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	applock "github.com/nevvesdev/distributed-lock-manager/internal/application/lock"
	infrahttp "github.com/nevvesdev/distributed-lock-manager/internal/infra/http"
	"github.com/nevvesdev/distributed-lock-manager/internal/infra/metrics"
	infraredis "github.com/nevvesdev/distributed-lock-manager/internal/infra/redis"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	ctx := context.Background()

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient, err := infraredis.NewClient(ctx, infraredis.Config{
		Addr:     redisAddr,
		PoolSize: 10,
	})
	if err != nil {
		slog.Error("falha ao conectar no Redis", "erro", err)
		os.Exit(1)
	}
	slog.Info("conectado ao Redis", "addr", redisAddr)

	m := metrics.New()
	repo := infraredis.NewLockRepository(redisClient)
	base := applock.NewLockService(repo)
	service := applock.NewInstrumentedLockService(base, m)

	router := infrahttp.NewRouter(service)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", router)

	addr := ":8080"
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	slog.Info("servidor iniciado", "addr", addr)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("servidor encerrado", "erro", err)
		os.Exit(1)
	}
}
