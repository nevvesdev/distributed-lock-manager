package lock

import "errors"

var (
	ErrLockAlreadyHeld      = errors.New("lock já está sendo mantido por outro processo")
	ErrLockNotFound         = errors.New("lock não encontrado")
	ErrFencingTokenMismatch = errors.New("fencing token inválido: possível operação de processo obsoleto")
	ErrLockNotOwned         = errors.New("lock não pertence a este processo")
)
