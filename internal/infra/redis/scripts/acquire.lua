-- KEYS[1] = chave do lock     (dlm:lock:<key>)
-- KEYS[2] = chave do token    (dlm:token:<key>)
-- ARGV[1] = owner
-- ARGV[2] = ttl em milissegundos

local lock_key  = KEYS[1]
local token_key = KEYS[2]
local owner     = ARGV[1]
local ttl_ms    = tonumber(ARGV[2])

local ok = redis.call("SET", lock_key, owner, "NX", "PX", ttl_ms)

if not ok then
  return -1
end

local token = redis.call("INCR", token_key)
return token