package fencing

// O contador é incrementado atomicamente a cada acquire, garantindo monotonicidade.
func TokenKey(lockKey string) string {
	return "dlm:token:" + lockKey
}

// LockKey retorna a chave Redis usada para armazenar os dados do lock.
func LockKey(lockKey string) string {
	return "dlm:lock:" + lockKey
}
