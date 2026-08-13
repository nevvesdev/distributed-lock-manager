package lock

import "time"

// Lock representa um lock distribuído adquirido por um processo.
type Lock struct {
	Key          string
	Owner        string
	FencingToken int64
	TTL          time.Duration
	AcquiredAt   time.Time
	ExpiresAt    time.Time
}

func (l *Lock) IsExpired() bool {
	return time.Now().After(l.ExpiresAt)
}

func (l *Lock) IsOwnedBy(owner string) bool {
	return l.Owner == owner
}
