package sqlite3

import (
	"context"
	"errors"
	"fmt"

	"github.com/er-davo/dberr"
	"modernc.org/sqlite"
)

func init() {
	dberr.Register(&sqliteDriver{})
}

func makeErr(msg string, code string, err error) error {
	return &dberr.Error{
		Msg:  msg,
		Code: code,
		Err:  err,
	}
}

type sqliteDriver struct{}

// SQLite primary result codes.
const (
	sqliteError      = 1
	sqliteInternal   = 2
	sqlitePerm       = 3
	sqliteAbort      = 4
	sqliteBusy       = 5
	sqliteLocked     = 6
	sqliteNoMem      = 7
	sqliteReadOnly   = 8
	sqliteInterrupt  = 9
	sqliteIOErr      = 10
	sqliteCorrupt    = 11
	sqliteFull       = 13
	sqliteCantOpen   = 14
	sqliteProtocol   = 15
	sqliteToobig     = 18
	sqliteConstraint = 19
	sqliteMismatch   = 20
	sqliteAuth       = 23
)

// SQLite extended result codes.
const (
	sqliteConstraintCheck      = 275
	sqliteConstraintForeignKey = 787
	sqliteConstraintNotNull    = 1299
	sqliteConstraintPrimaryKey = 1555
	sqliteConstraintUnique     = 2067
	sqliteConstraintDataType   = 3091

	sqliteAbortRollback = 516

	sqliteBusyRecovery = 261
	sqliteBusySnapshot = 517
	sqliteBusyTimeout  = 773

	sqliteLockedSharedCache = 262

	sqliteReadOnlyRecovery = 264
	sqliteReadOnlyCantLock = 520
	sqliteReadOnlyRollback = 776
	sqliteReadOnlyCantInit = 1288
)

func (s *sqliteDriver) Wrap(err error) error {
	if err == nil {
		return nil
	}

	// Context errors should be handled before SQLite-specific errors.
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

	var sqlErr *sqlite.Error
	if !errors.As(err, &sqlErr) {
		return makeErr(
			"internal database error",
			"internal",
			errors.Join(dberr.ErrInternal, err),
		)
	}

	code := sqlErr.Code()
	codeStr := fmt.Sprintf("%d", code)

	return s.wrapSQLiteError(err, code, codeStr)
}

func (s *sqliteDriver) wrapSQLiteError(
	original error,
	code int,
	codeStr string,
) error {
	switch code {

	// -------------------------------------------------------------------------
	// Conflicts / constraints
	// -------------------------------------------------------------------------

	case sqliteConstraintUnique:
		conflictErr := &dberr.ConflictError{
			Type: dberr.ConflictUnique,
			Err:  original,
		}

		return makeErr(
			"unique constraint violation",
			codeStr,
			conflictErr,
		)

	case sqliteConstraintPrimaryKey:
		conflictErr := &dberr.ConflictError{
			Type: dberr.ConflictUnique,
			Err:  original,
		}

		return makeErr(
			"primary key violation",
			codeStr,
			conflictErr,
		)

	case sqliteConstraintForeignKey:
		fkErr := &dberr.ForeignKeyError{
			Err: original,
		}

		return makeErr(
			"foreign key violation",
			codeStr,
			fkErr,
		)

	case sqliteConstraintNotNull:
		validationErr := &dberr.ValidationError{
			Type: dberr.ValidationNotNull,
			Err:  original,
		}

		return makeErr(
			"not-null constraint violation",
			codeStr,
			validationErr,
		)

	case sqliteConstraintCheck:
		validationErr := &dberr.ValidationError{
			Type: dberr.ValidationCheck,
			Err:  original,
		}

		return makeErr(
			"check constraint violation",
			codeStr,
			validationErr,
		)

	case sqliteConstraintDataType:
		validationErr := &dberr.ValidationError{
			Type: dberr.ValidationTypeMismatch,
			Err:  original,
		}

		return makeErr(
			"database type constraint violation",
			codeStr,
			validationErr,
		)

	// Generic SQLITE_CONSTRAINT.
	case sqliteConstraint:
		validationErr := &dberr.ValidationError{
			Type: dberr.ValidationCheck,
			Err:  original,
		}

		return makeErr(
			"database constraint violation",
			codeStr,
			validationErr,
		)

	// -------------------------------------------------------------------------
	// Transactions
	// -------------------------------------------------------------------------

	case sqliteAbortRollback:
		txErr := &dberr.TransactionError{
			Type: dberr.TransactionAborted,
			Err:  original,
		}

		return makeErr(
			"transaction was rolled back",
			codeStr,
			errors.Join(txErr, dberr.ErrTransaction),
		)

	case sqliteBusySnapshot:
		txErr := &dberr.TransactionError{
			Type: dberr.TransactionSerialization,
			Err:  original,
		}

		return makeErr(
			"transaction serialization failure",
			codeStr,
			errors.Join(txErr, dberr.ErrRetryable),
		)

	case sqliteBusyRecovery:
		return makeErr(
			"database is busy recovering",
			codeStr,
			errors.Join(dberr.ErrRetryable, original),
		)

	case sqliteBusyTimeout:
		return makeErr(
			"database lock acquisition timed out",
			codeStr,
			errors.Join(dberr.ErrTimeout, dberr.ErrRetryable, original),
		)

	// SQLITE_BUSY.
	//
	// SQLite documents that BUSY can occur because another connection holds
	// the required lock. A retry may be appropriate, but if this occurs
	// inside an explicit transaction the transaction may need to be rolled
	// back first.
	case sqliteBusy:
		return makeErr(
			"database is busy",
			codeStr,
			errors.Join(dberr.ErrRetryable, original),
		)

	case sqliteLocked, sqliteLockedSharedCache:
		return makeErr(
			"database is locked",
			codeStr,
			errors.Join(dberr.ErrRetryable, original),
		)

	// -------------------------------------------------------------------------
	// Read-only / permissions
	// -------------------------------------------------------------------------

	case sqliteReadOnly,
		sqliteReadOnlyRecovery,
		sqliteReadOnlyCantLock,
		sqliteReadOnlyRollback,
		sqliteReadOnlyCantInit:
		return makeErr(
			"database is read-only",
			codeStr,
			errors.Join(dberr.ErrReadOnly, original),
		)

	case sqlitePerm, sqliteAuth:
		return makeErr(
			"database permission denied",
			codeStr,
			errors.Join(dberr.ErrPermission, original),
		)

	// -------------------------------------------------------------------------
	// Cancellation
	// -------------------------------------------------------------------------

	case sqliteInterrupt:
		return makeErr(
			"database operation canceled",
			codeStr,
			errors.Join(dberr.ErrCanceled, original),
		)

	// -------------------------------------------------------------------------
	// Connection / opening
	// -------------------------------------------------------------------------

	case sqliteCantOpen:
		return makeErr(
			"database connection could not be opened",
			codeStr,
			errors.Join(dberr.ErrConnection, original),
		)

	// -------------------------------------------------------------------------
	// Internal / infrastructure errors
	// -------------------------------------------------------------------------

	case sqliteNoMem:
		return makeErr(
			"database memory allocation failed",
			codeStr,
			errors.Join(dberr.ErrInternal, original),
		)

	case sqliteCorrupt:
		return makeErr(
			"database is corrupted",
			codeStr,
			errors.Join(dberr.ErrInternal, original),
		)

	case sqliteIOErr:
		return makeErr(
			"database I/O error",
			codeStr,
			errors.Join(dberr.ErrInternal, original),
		)

	case sqliteProtocol:
		return makeErr(
			"database protocol error",
			codeStr,
			errors.Join(dberr.ErrInternal, original),
		)

	case sqliteToobig:
		validationErr := &dberr.ValidationError{
			Type: dberr.ValidationOutOfRange,
			Err:  original,
		}

		return makeErr(
			"database value is too large",
			codeStr,
			validationErr,
		)

	case sqliteMismatch:
		validationErr := &dberr.ValidationError{
			Type: dberr.ValidationTypeMismatch,
			Err:  original,
		}

		return makeErr(
			"database type mismatch",
			codeStr,
			validationErr,
		)

	case sqliteInternal:
		return makeErr(
			"internal database error",
			codeStr,
			errors.Join(dberr.ErrInternal, original),
		)

	case sqliteError:
		return makeErr(
			"database error",
			codeStr,
			errors.Join(dberr.ErrInternal, original),
		)

	default:
		return makeErr(
			"internal database error",
			codeStr,
			errors.Join(dberr.ErrInternal, original),
		)
	}
}
