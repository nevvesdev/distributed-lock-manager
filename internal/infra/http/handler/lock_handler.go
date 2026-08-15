package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	applock "github.com/nevvesdev/distributed-lock-manager/internal/application/lock"
	domlock "github.com/nevvesdev/distributed-lock-manager/internal/domain/lock"
)

// expõe operações de lock distribuído via HTTP.
type LockHandler struct {
	service *applock.LockService
}

// cria um novo LockHandler com o serviço fornecido.
func NewLockHandler(service *applock.LockService) *LockHandler {
	return &LockHandler{service: service}
}

// representa o corpo da requisição de acquire.
type acquireRequest struct {
	Owner string `json:"owner"`
	TTL   string `json:"ttl"`
}

// representa a resposta de uma operação de lock.
type lockResponse struct {
	Key          string    `json:"key"`
	Owner        string    `json:"owner"`
	FencingToken int64     `json:"fencing_token"`
	TTL          string    `json:"ttl"`
	AcquiredAt   time.Time `json:"acquired_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// representa o corpo da requisição de release.
type releaseRequest struct {
	Owner        string `json:"owner"`
	FencingToken int64  `json:"fencing_token"`
}

// renewRequest representa o corpo da requisição de renew.
type renewRequest struct {
	Owner        string `json:"owner"`
	FencingToken int64  `json:"fencing_token"`
	TTL          string `json:"ttl"`
}

// errorResponse representa uma resposta de erro.
type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

// POST /locks/{key}/acquire
func (h *LockHandler) Acquire(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	var req acquireRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}

	if req.Owner == "" {
		writeError(w, http.StatusBadRequest, "campo 'owner' é obrigatório")
		return
	}

	ttl, err := time.ParseDuration(req.TTL)
	if err != nil || ttl <= 0 {
		writeError(w, http.StatusBadRequest, "campo 'ttl' inválido (ex: '30s', '1m')")
		return
	}

	l, err := h.service.Acquire(r.Context(), key, req.Owner, ttl)
	if err != nil {
		if errors.Is(err, domlock.ErrLockAlreadyHeld) {
			writeError(w, http.StatusConflict, "lock já está sendo mantido por outro processo")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao adquirir lock")
		return
	}

	w.Header().Set("X-Fencing-Token", strconv.FormatInt(l.FencingToken, 10))
	writeJSON(w, http.StatusCreated, lockResponse{
		Key:          l.Key,
		Owner:        l.Owner,
		FencingToken: l.FencingToken,
		TTL:          l.TTL.String(),
		AcquiredAt:   l.AcquiredAt,
		ExpiresAt:    l.ExpiresAt,
	})
}

// DELETE /locks/{key}/release
func (h *LockHandler) Release(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	var req releaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}

	if req.Owner == "" || req.FencingToken == 0 {
		writeError(w, http.StatusBadRequest, "campos 'owner' e 'fencing_token' são obrigatórios")
		return
	}

	err := h.service.Release(r.Context(), key, req.Owner, req.FencingToken)
	if err != nil {
		switch {
		case errors.Is(err, domlock.ErrLockNotFound):
			writeError(w, http.StatusNotFound, "lock não encontrado")
		case errors.Is(err, domlock.ErrLockNotOwned):
			writeError(w, http.StatusForbidden, "lock não pertence a este processo")
		case errors.Is(err, domlock.ErrFencingTokenMismatch):
			writeError(w, http.StatusConflict, "fencing token inválido: possível processo obsoleto")
		default:
			writeError(w, http.StatusInternalServerError, "erro ao liberar lock")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /locks/{key}
func (h *LockHandler) Get(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	l, err := h.service.Get(r.Context(), key)
	if err != nil {
		if errors.Is(err, domlock.ErrLockNotFound) {
			writeError(w, http.StatusNotFound, "lock não encontrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro ao buscar lock")
		return
	}

	writeJSON(w, http.StatusOK, lockResponse{
		Key:          l.Key,
		Owner:        l.Owner,
		FencingToken: l.FencingToken,
		TTL:          l.TTL.String(),
		AcquiredAt:   l.AcquiredAt,
		ExpiresAt:    l.ExpiresAt,
	})
}

// POST /locks/{key}/renew
func (h *LockHandler) Renew(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	var req renewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "corpo da requisição inválido")
		return
	}

	if req.Owner == "" || req.FencingToken == 0 {
		writeError(w, http.StatusBadRequest, "campos 'owner' e 'fencing_token' são obrigatórios")
		return
	}

	ttl, err := time.ParseDuration(req.TTL)
	if err != nil || ttl <= 0 {
		writeError(w, http.StatusBadRequest, "campo 'ttl' inválido (ex: '30s', '1m')")
		return
	}

	err = h.service.Renew(r.Context(), key, req.Owner, req.FencingToken, ttl)
	if err != nil {
		switch {
		case errors.Is(err, domlock.ErrLockNotFound):
			writeError(w, http.StatusNotFound, "lock não encontrado ou expirado")
		case errors.Is(err, domlock.ErrLockNotOwned):
			writeError(w, http.StatusForbidden, "lock não pertence a este processo")
		case errors.Is(err, domlock.ErrFencingTokenMismatch):
			writeError(w, http.StatusConflict, "fencing token inválido: possível processo obsoleto")
		default:
			writeError(w, http.StatusInternalServerError, "erro ao renovar lock")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
