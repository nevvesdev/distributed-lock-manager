-- KEYS[1] = chave do lock     (dlm:lock:<key>)
-- KEYS[2] = chave do token    (dlm:token:<key>)
-- ARGV[1] = owner
-- ARGV[2] = fencing token esperado
-- ARGV[3] = novo ttl em milissegundos

-- Retorna:
--   1  = TTL renovado com sucesso
--   0  = lock não existe (expirou antes da renovação)
--  -1  = owner não corresponde
--  -2  = fencing token não corresponde (processo obsoleto)

local lock_key     = KEYS[1]
local token_key    = KEYS[2]
local owner        = ARGV[1]
local expected_tok = tonumber(ARGV[2])
local ttl_ms       = tonumber(ARGV[3])

local current_owner = redis.call("GET", lock_key)

if current_owner == false then
  return 0
end

if current_owner ~= owner then
  return -1
end

local current_token = tonumber(redis.call("GET", token_key))

if current_token ~= expected_tok then
  return -2
end

redis.call("PEXPIRE", lock_key, ttl_ms)

return 1