package dberr

import "errors"

var (
	// Resource state.

	// ErrNotFound indicates that the requested resource does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrConflict indicates a conflict with the current state of a resource,
	// such as a unique or exclusion constraint violation.
	ErrConflict = errors.New("resource conflict")

	// Data integrity.

	// ErrForeignKey indicates a foreign key constraint violation.
	ErrForeignKey = errors.New("foreign key violation")

	// ErrValidation indicates that the provided data violates a database
	// constraint or cannot be represented by the database.
	ErrValidation = errors.New("database validation failed")

	// Query execution.

	// ErrNoRowsAffected indicates that an UPDATE or DELETE operation
	// did not affect any rows.
	ErrNoRowsAffected = errors.New("no rows affected")

	// ErrCanceled indicates that the database operation was canceled.
	ErrCanceled = errors.New("database operation canceled")

	// ErrTimeout indicates that the database operation exceeded its deadline.
	ErrTimeout = errors.New("database operation timed out")

	// Transaction.

	// ErrTransaction indicates a transaction-related error.
	ErrTransaction = errors.New("database transaction error")

	// ErrTransactionClosed indicates that the transaction is already closed.
	ErrTransactionClosed = errors.New("database transaction is closed")

	// ErrTransactionAborted indicates that the transaction has been aborted
	// and cannot continue.
	ErrTransactionAborted = errors.New("database transaction is aborted")

	// ErrRetryable indicates that the operation may succeed if retried.
	ErrRetryable = errors.New("database operation is retryable")

	// Connection and infrastructure.

	// ErrConnection indicates that the database connection could not be
	// established or is no longer usable.
	ErrConnection = errors.New("database connection error")

	// ErrPermission indicates that the database user does not have sufficient
	// privileges to perform the operation.
	ErrPermission = errors.New("database permission denied")

	// ErrReadOnly indicates that the database or current transaction is
	// read-only and the requested operation requires a write.
	ErrReadOnly = errors.New("database is read-only")

	// Other.

	// ErrInternal indicates an unexpected database or driver error that cannot
	// be classified more specifically.
	ErrInternal = errors.New("internal database error")
)
