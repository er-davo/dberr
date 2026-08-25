package dberr

type TransactionType string

const (
	TransactionClosed        TransactionType = "closed"
	TransactionAborted       TransactionType = "aborted"
	TransactionDeadlock      TransactionType = "deadlock"
	TransactionSerialization TransactionType = "serialization"
)

type TransactionError struct {
	Type TransactionType
	Err  error
}

func (e *TransactionError) Error() string {
	return "database transaction error"
}

func (e *TransactionError) Unwrap() error {
	return e.Err
}

func (e *TransactionError) Is(target error) bool {
	switch target {
	case ErrTransaction:
		return true
	case ErrTransactionClosed:
		return e.Type == TransactionClosed
	case ErrTransactionAborted:
		return e.Type == TransactionAborted
	default:
		return false
	}
}
