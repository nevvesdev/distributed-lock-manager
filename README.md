# distributed-lock-manager

Gerenciador de locks distribuídos implementado em Go com Redis, scripts Lua atômicos, fencing tokens e renovação automática via heartbeat.

Projeto de portfólio focado em sistemas distribuídos — aborda o bug clássico de split-brain onde um lock expira mas o processo original ainda age como se o possuísse.

Sem fencing tokens, o Processo A causaria corrupção de dados. Com fencing tokens, qualquer operação com token desatualizado é rejeitada automaticamente.

---

## Como fencing tokens resolvem o problema

Cada `acquire` incrementa atomicamente um contador no Redis via script Lua. O token retornado é monotônico crescente — nunca decresce, mesmo após expiração e re-acquire.

O recurso protegido (banco, arquivo, API) deve rejeitar operações cujo token seja menor que o último token visto. Isso garante que processos "zumbis" nunca consigam agir.

---

## Por que scripts Lua garantem atomicidade

Operações como `SET NX + INCR` precisam ser atômicas. Se executadas separadamente, uma falha entre elas deixaria o sistema em estado inconsistente. Scripts Lua no Redis são executados atomicamente — nenhuma outra operação pode interromper um script em execução.

---

## Stack

- **Go 1.22+**
- **Redis 7.2** — armazenamento do lock e contador de fencing token
- **Lua scripts** — atomicidade garantida para acquire, release e renew
- **Prometheus** — métricas de acquire, release, renew, contenção e duração
- **Cobra CLI** — interface de linha de comando para operações manuais
- **chi** — roteador HTTP leve para a API de demonstração
- **miniredis** — Redis em memória para testes sem dependências externas

---

## HTTP API

```bash
# adquire lock
curl -X POST http://localhost:8080/locks/{key}/acquire \
  -H "Content-Type: application/json" \
  -d '{"owner":"worker-A","ttl":"30s"}'

# consulta estado
curl http://localhost:8080/locks/{key}

# renova TTL manualmente
curl -X POST http://localhost:8080/locks/{key}/renew \
  -H "Content-Type: application/json" \
  -d '{"owner":"worker-A","fencing_token":1,"ttl":"30s"}'

# libera lock
curl -X DELETE http://localhost:8080/locks/{key}/release \
  -H "Content-Type: application/json" \
  -d '{"owner":"worker-A","fencing_token":1}'

# demonstração do split-brain
curl -X POST http://localhost:8080/demo/split-brain
```

O fencing token é retornado no body e no header `X-Fencing-Token`.


