# distributed-lock-manager

Gerenciador de locks distribuídos com Redis, Lua scripts atômicos, fencing tokens e renovação automática via heartbeat.

---

Fencing tokens resolvem isso: cada acquire gera um token monotônico crescente.
O recurso protegido rejeita operações com token menor que o último visto.

## Stack

- Go 1.22+
- Redis 7.2
- Lua scripts (atomicidade garantida)
- Prometheus metrics
- Cobra CLI