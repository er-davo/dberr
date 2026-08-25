package pgx

import (
	"context"
	"errors"

	"github.com/er-davo/dberr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func init() {
	dberr.Register(&pgxDriver{})
}

func makeErr(msg, code string, err error) error {
	return &dberr.Error{
		Msg:  msg,
		Code: code,
		Err:  err,
	}
}

const (
	// Class 08 — Connection Exception.
	pgConnectionException          = "08000"
	pgConnectionDoesNotExist       = "08003"
	pgConnectionFailure            = "08006"
	pgUnableToEstablish            = "08001"
	pgTransactionResolutionUnknown = "08007"
	pgProtocolViolation            = "08P01"

	// Class 22 — Data Exception.
	pgStringDataRightTruncation   = "22001"
	pgNumericValueOutOfRange      = "22003"
	pgInvalidTextRepresentation   = "22P02"
	pgInvalidBinaryRepresentation = "22P03"

	// Class 23 — Integrity Constraint Violation.
	pgRestrictViolation   = "23001"
	pgNotNullViolation    = "23502"
	pgForeignKeyViolation = "23503"
	pgUniqueViolation     = "23505"
	pgCheckViolation      = "23514"
	pgExclusionViolation  = "23P01"

	// Class 25 — Invalid Transaction State.
	pgInvalidTransactionState = "25000"
	pgReadOnlyTransaction     = "25006"
	pgInFailedTransaction     = "25P02"

	// Class 40 — Transaction Rollback.
	pgTransactionRollback        = "40000"
	pgTransactionIntegrity       = "40002"
	pgSerializationFailure       = "40001"
	pgStatementCompletionUnknown = "40003"
	pgDeadlockDetected           = "40P01"

	// Class 42 — Syntax Error or Access Rule Violation.
	pgInsufficientPrivilege = "42501"
	pgInvalidForeignKey     = "42830"

	// Class 57 — Operator Intervention.
	pgQueryCanceled = "57014"
)

type pgxDriver struct{}

func (p *pgxDriver) Wrap(err error) error {
	if err == nil {
		return nil
	}

	// Context errors should be checked before PgError.
	if errors.Is(err, context.Canceled) {
		return makeErr(
			"database operation canceled",
			"canceled",
			errors.Join(dberr.ErrCanceled, err),
		)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return makeErr(
			"database operation timed out",
			"timeout",
			errors.Join(dberr.ErrTimeout, err),
		)
	}

	// pgx-specific errors.

	if errors.Is(err, pgx.ErrNoRows) {
		return makeErr(
			"record not found",
			"no_rows",
			errors.Join(dberr.ErrNotFound, err),
		)
	}

	if errors.Is(err, pgx.ErrTxClosed) {
		txErr := &dberr.TransactionError{
			Type: dberr.TransactionClosed,
			Err:  err,
		}

		return makeErr(
			"transaction is closed",
			"tx_closed",
			errors.Join(txErr, dberr.ErrTransaction),
		)
	}

	if errors.Is(err, pgx.ErrTxCommitRollback) {
		txErr := &dberr.TransactionError{
			Type: dberr.TransactionAborted,
			Err:  err,
		}

		return makeErr(
			"transaction was rolled back",
			"tx_commit_rollback",
			errors.Join(txErr, dberr.ErrTransaction),
		)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return p.wrapPgError(err, pgErr)
	}

	// Unknown driver / connection error.
	return makeErr(
		"internal database error",
		"internal",
		errors.Join(dberr.ErrInternal, err),
	)
}

func (p *pgxDriver) wrapPgError(original error, pgErr *pgconn.PgError) error {
	switch pgErr.Code {
	case pgUniqueViolation:
		conflictErr := &dberr.ConflictError{
			Type:       dberr.ConflictUnique,
			Table:      pgErr.TableName,
			Column:     pgErr.ColumnName,
			Constraint: pgErr.ConstraintName,
			Err:        original,
		}

		return makeErr(
			"unique constraint violation",
			pgErr.Code,
			conflictErr,
		)

	case pgExclusionViolation:
		conflictErr := &dberr.ConflictError{
			Type:       dberr.ConflictExclusion,
			Table:      pgErr.TableName,
			Column:     pgErr.ColumnName,
			Constraint: pgErr.ConstraintName,
			Err:        original,
		}

		return makeErr(
			"exclusion constraint violation",
			pgErr.Code,
			conflictErr,
		)

	case pgForeignKeyViolation:
		fkErr := &dberr.ForeignKeyError{
			Table:      pgErr.TableName,
			Column:     pgErr.ColumnName,
			Constraint: pgErr.ConstraintName,
			Err:        original,
		}

		return makeErr(
			"foreign key violation",
			pgErr.Code,
			fkErr,
		)

	case pgNotNullViolation:
		validationErr := &dberr.ValidationError{
			Type:       dberr.ValidationNotNull,
			Table:      pgErr.TableName,
			Column:     pgErr.ColumnName,
			Constraint: pgErr.ConstraintName,
			Err:        original,
		}

		return makeErr(
			"not-null constraint violation",
			pgErr.Code,
			validationErr,
		)

	case pgCheckViolation:
		validationErr := &dberr.ValidationError{
			Type:       dberr.ValidationCheck,
			Table:      pgErr.TableName,
			Column:     pgErr.ColumnName,
			Constraint: pgErr.ConstraintName,
			Err:        original,
		}

		return makeErr(
			"check constraint violation",
			pgErr.Code,
			validationErr,
		)

	case pgStringDataRightTruncation,
		pgNumericValueOutOfRange,
		pgInvalidTextRepresentation,
		pgInvalidBinaryRepresentation:
		validationErr := &dberr.ValidationError{
			Type: dberr.ValidationInvalidFormat,
			Err:  original,
		}

		return makeErr(
			"invalid database value",
			pgErr.Code,
			validationErr,
		)

	case pgSerializationFailure:
		txErr := &dberr.TransactionError{
			Type: dberr.TransactionSerialization,
			Err:  original,
		}

		return makeErr(
			"transaction serialization failure",
			pgErr.Code,
			errors.Join(txErr, dberr.ErrRetryable),
		)

	case pgDeadlockDetected:
		txErr := &dberr.TransactionError{
			Type: dberr.TransactionDeadlock,
			Err:  original,
		}

		return makeErr(
			"transaction deadlock detected",
			pgErr.Code,
			errors.Join(txErr, dberr.ErrRetryable),
		)

	case pgReadOnlyTransaction:
		return makeErr(
			"transaction is read-only",
			pgErr.Code,
			errors.Join(dberr.ErrReadOnly, original),
		)

	case pgInFailedTransaction,
		pgInvalidTransactionState:
		txErr := &dberr.TransactionError{
			Type: dberr.TransactionAborted,
			Err:  original,
		}

		return makeErr(
			"transaction is aborted",
			pgErr.Code,
			errors.Join(txErr, dberr.ErrTransaction),
		)

	case pgInsufficientPrivilege:
		return makeErr(
			"database permission denied",
			pgErr.Code,
			errors.Join(dberr.ErrPermission, original),
		)

	case pgQueryCanceled:
		return makeErr(
			"database query canceled",
			pgErr.Code,
			errors.Join(dberr.ErrCanceled, original),
		)

	case pgConnectionException,
		pgConnectionDoesNotExist,
		pgConnectionFailure,
		pgUnableToEstablish,
		pgTransactionResolutionUnknown,
		pgProtocolViolation:
		return makeErr(
			"database connection error",
			pgErr.Code,
			errors.Join(dberr.ErrConnection, original),
		)

	case pgInvalidForeignKey:
		validationErr := &dberr.ValidationError{
			Type: dberr.ValidationCheck,
			Err:  original,
		}

		return makeErr(
			"invalid foreign key definition",
			pgErr.Code,
			validationErr,
		)

	case pgRestrictViolation:
		return makeErr(
			"referenced resource cannot be modified",
			pgErr.Code,
			errors.Join(dberr.ErrForeignKey, original),
		)

	case pgTransactionRollback,
		pgTransactionIntegrity,
		pgStatementCompletionUnknown:
		txErr := &dberr.TransactionError{
			Type: dberr.TransactionAborted,
			Err:  original,
		}

		return makeErr(
			"transaction rollback",
			pgErr.Code,
			errors.Join(txErr, dberr.ErrTransaction),
		)

	default:
		return makeErr(
			"internal database error",
			pgErr.Code,
			errors.Join(dberr.ErrInternal, original),
		)
	}
}
