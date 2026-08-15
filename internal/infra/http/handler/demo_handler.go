package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	applock "github.com/nevvesdev/distributed-lock-manager/internal/application/lock"
)

// DemoHandler expõe endpoints de demonstração do comportamento do fencing token.
type DemoHandler struct {
	service applock.Service
}

// NewDemoHandler cria um novo DemoHandler.
func NewDemoHandler(service applock.Service) *DemoHandler {
	return &DemoHandler{service: service}
}

// stepEvent representa um passo da simulação de split-brain.
type stepEvent struct {
	Passo     int    `json:"passo"`
	Ator      string `json:"ator"`
	Acao      string `json:"acao"`
	Token     int64  `json:"token,omitempty"`
	Resultado string `json:"resultado"`
}

// splitBrainResponse é a resposta completa da simulação.
type splitBrainResponse struct {
	Descricao string      `json:"descricao"`
	Passos    []stepEvent `json:"passos"`
	Conclusao string      `json:"conclusao"`
}

// SplitBrain godoc
// POST /demo/split-brain
//
// Simula o bug clássico de split-brain em sistemas distribuídos:
//  1. Processo A adquire o lock (token=N)
//  2. Processo A sofre GC pause (simulado com time.Sleep)
//  3. Lock expira no Redis
//  4. Processo B adquire o lock expirado (token=N+1)
//  5. Processo B opera com segurança
//  6. Processo A "acorda" e tenta operar — bloqueado pelo token mismatch
func (h *DemoHandler) SplitBrain(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	key := "demo:split-brain"
	steps := make([]stepEvent, 0, 6)

	// limpa estado anterior para a demo ser idempotente
	_ = h.service.Release(ctx, key, "worker-A", 1)
	_ = h.service.Release(ctx, key, "worker-B", 2)

	// Passo 1 — Processo A adquire o lock com TTL curto
	lockA, err := h.service.Acquire(ctx, key, "worker-A", 500*time.Millisecond)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "falha ao adquirir lock para worker-A: "+err.Error())
		return
	}
	steps = append(steps, stepEvent{
		Passo:     1,
		Ator:      "worker-A",
		Acao:      "adquire o lock com TTL de 500ms",
		Token:     lockA.FencingToken,
		Resultado: fmt.Sprintf("sucesso — lock adquirido com token=%d", lockA.FencingToken),
	})

	// Passo 2 — Processo A sofre GC pause
	steps = append(steps, stepEvent{
		Passo:     2,
		Ator:      "worker-A",
		Acao:      "sofre GC pause / rede lenta (sleep de 600ms)",
		Token:     lockA.FencingToken,
		Resultado: "processo congelado — não renova o lock",
	})
	time.Sleep(600 * time.Millisecond)

	// Passo 3 — Lock expirou
	steps = append(steps, stepEvent{
		Passo:     3,
		Ator:      "Redis",
		Acao:      "TTL do lock expirou após 500ms",
		Resultado: "lock removido automaticamente pelo Redis",
	})

	// Passo 4 — Processo B adquire o lock expirado
	lockB, err := h.service.Acquire(ctx, key, "worker-B", 30*time.Second)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "falha ao adquirir lock para worker-B: "+err.Error())
		return
	}
	steps = append(steps, stepEvent{
		Passo:     4,
		Ator:      "worker-B",
		Acao:      "adquire o lock que expirou",
		Token:     lockB.FencingToken,
		Resultado: fmt.Sprintf("sucesso — lock adquirido com token=%d (maior que o anterior)", lockB.FencingToken),
	})

	// Passo 5 — Processo B opera normalmente
	steps = append(steps, stepEvent{
		Passo:     5,
		Ator:      "worker-B",
		Acao:      fmt.Sprintf("executa operação crítica com token=%d", lockB.FencingToken),
		Token:     lockB.FencingToken,
		Resultado: "operação executada com segurança",
	})

	// Passo 6 — Processo A "acorda" e tenta liberar com token antigo
	errRelease := h.service.Release(ctx, key, "worker-A", lockA.FencingToken)
	var resultado string
	if errRelease != nil {
		resultado = fmt.Sprintf(
			"BLOQUEADO — %s (token=%d é obsoleto, token atual=%d)",
			errRelease.Error(), lockA.FencingToken, lockB.FencingToken,
		)
	} else {
		resultado = "FALHA NA PROTEÇÃO — worker-A conseguiu liberar o lock de worker-B (bug!)"
	}
	steps = append(steps, stepEvent{
		Passo:     6,
		Ator:      "worker-A",
		Acao:      fmt.Sprintf("acorda e tenta liberar o lock com token antigo=%d", lockA.FencingToken),
		Token:     lockA.FencingToken,
		Resultado: resultado,
	})

	// limpa ao final
	_ = h.service.Release(ctx, key, "worker-B", lockB.FencingToken)

	writeJSON(w, http.StatusOK, splitBrainResponse{
		Descricao: "Simulação do bug clássico de split-brain em locks distribuídos. " +
			"O fencing token garante que processos obsoletos não consigam operar após o lock expirar.",
		Passos: steps,
		Conclusao: "Fencing tokens resolvem o split-brain: cada acquire gera um token monotônico crescente. " +
			"O recurso protegido rejeita operações com token menor que o último visto, " +
			"bloqueando processos que acordaram tarde demais.",
	})
}
