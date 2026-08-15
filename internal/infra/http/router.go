package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	applock "github.com/nevvesdev/distributed-lock-manager/internal/application/lock"
	"github.com/nevvesdev/distributed-lock-manager/internal/infra/http/handler"
)

// monta o roteador HTTP com todos os endpoints de lock.
func NewRouter(service applock.Service) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	h := handler.NewLockHandler(service)

	r.Post("/locks/{key}/acquire", h.Acquire)
	r.Delete("/locks/{key}/release", h.Release)
	r.Get("/locks/{key}", h.Get)
	r.Post("/locks/{key}/renew", h.Renew)

	return r
}
