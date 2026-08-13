package fencing

func TokenKey(lockKey string) string {
	return "dlm:token:" + lockKey
}

func LockKey(lockKey string) string {
	return "dlm:lock:" + lockKey
}
