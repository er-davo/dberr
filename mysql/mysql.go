package mysql

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/er-davo/dberr"
	gomysql "github.com/go-sql-driver/mysql"
)

func init() {
	dberr.Register(&mysqlDriver{})
}

func makeErr(msg, code string, err error) error {
	return &dberr.Error{
		Msg:  msg,
		Code: code,
		Err:  err,
	}
}

type mysqlDriver struct{}

// MySQL server error codes.
const (
	// Constraints.
	mysqlDuplicateEntry           = 1062
	mysqlDuplicateKeyName         = 1061
	mysqlNoReferencedRow          = 1216
	mysqlRowIsReferenced          = 1217
	mysqlCannotAddForeignKey      = 1215
	mysqlRowIsReferenced2         = 1451
	mysqlNoReferencedRow2         = 1452
	mysqlBadNullError             = 1048
	mysqlCheckConstraintViolation = 3819

	// Transactions / locking.
	mysqlLockWaitTimeout = 1205
	mysqlDeadlock        = 1213
	mysqlReadOnlyTx      = 1207

	// Permissions / access.
	mysqlAccessDenied        = 1044
	mysqlAccessDeniedForUser = 1045
	mysqlTableAccessDenied   = 1142
	mysqlColumnAccessDenied  = 1143
	mysqlDBAccessDenied      = 1044

	// Connection / server.
	mysqlTooManyConnections = 1040
	mysqlServerGone         = 2006
	mysqlServerLost         = 2013

	// Read-only.
	mysqlReadOnly = 1290
)

func (m *mysqlDriver) Wrap(err error) error {
	if err == nil {
		return nil
	}

	// Generic context errors.
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

	// MySQL driver errors.
	var mysqlErr *gomysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return m.wrapMySQLError(err, mysqlErr)
	}

	// Driver-level connection errors.
	if errors.Is(err, gomysql.ErrInvalidConn) {
		return makeErr(
			"database connection is invalid",
			"invalid_connection",
			errors.Join(dberr.ErrConnection, err),
		)
	}

	return makeErr(
		"internal database error",
		"internal",
		errors.Join(dberr.ErrInternal, err),
	)
}

func (m *mysqlDriver) wrapMySQLError(
	original error,
	mysqlErr *gomysql.MySQLError,
) error {
	code := mysqlErr.Number
	codeStr := fmt.Sprintf("%d", code)

	switch code {

	// -------------------------------------------------------------------------
	// Conflict
	// -------------------------------------------------------------------------

	case mysqlDuplicateEntry:
		conflictErr := &dberr.ConflictError{
			Type:       dberr.ConflictUnique,
			Constraint: extractConstraint(mysqlErr.Message),
			Err:        original,
		}

		return makeErr(
			"unique constraint violation",
			codeStr,
			conflictErr,
		)

	case mysqlDuplicateKeyName:
		conflictErr := &dberr.ConflictError{
			Type: dberr.ConflictUnique,
			Err:  original,
		}

		return makeErr(
			"duplicate key name",
			codeStr,
			conflictErr,
		)

	// -------------------------------------------------------------------------
	// Foreign key
	// -------------------------------------------------------------------------

	case mysqlNoReferencedRow,
		mysqlRowIsReferenced,
		mysqlCannotAddForeignKey,
		mysqlRowIsReferenced2,
		mysqlNoReferencedRow2:

		fkErr := &dberr.ForeignKeyError{
			Err: original,
		}

		return makeErr(
			"foreign key violation",
			codeStr,
			fkErr,
		)

	// -------------------------------------------------------------------------
	// Validation
	// -------------------------------------------------------------------------

	case mysqlBadNullError:
		validationErr := &dberr.ValidationError{
			Type: dberr.ValidationNotNull,
			Err:  original,
		}

		return makeErr(
			"not-null constraint violation",
			codeStr,
			validationErr,
		)

	case mysqlCheckConstraintViolation:
		validationErr := &dberr.ValidationError{
			Type: dberr.ValidationCheck,
			Err:  original,
		}

		return makeErr(
			"check constraint violation",
			codeStr,
			validationErr,
		)

	// -------------------------------------------------------------------------
	// Transactions / locking
	// -------------------------------------------------------------------------

	case mysqlDeadlock:
		txErr := &dberr.TransactionError{
			Type: dberr.TransactionDeadlock,
			Err:  original,
		}

		return makeErr(
			"transaction deadlock detected",
			codeStr,
			errors.Join(txErr, dberr.ErrRetryable),
		)

	case mysqlLockWaitTimeout:
		return makeErr(
			"database lock wait timed out",
			codeStr,
			errors.Join(
				dberr.ErrTimeout,
				dberr.ErrRetryable,
				original,
			),
		)

	case mysqlReadOnlyTx:
		return makeErr(
			"transaction is read-only",
			codeStr,
			errors.Join(dberr.ErrReadOnly, original),
		)

	case mysqlReadOnly:
		return makeErr(
			"database is read-only",
			codeStr,
			errors.Join(dberr.ErrReadOnly, original),
		)

	// -------------------------------------------------------------------------
	// Permissions
	// -------------------------------------------------------------------------

	case mysqlAccessDenied,
		mysqlAccessDeniedForUser,
		mysqlTableAccessDenied,
		mysqlColumnAccessDenied:

		return makeErr(
			"database permission denied",
			codeStr,
			errors.Join(dberr.ErrPermission, original),
		)

	// -------------------------------------------------------------------------
	// Connection
	// -------------------------------------------------------------------------

	case mysqlTooManyConnections,
		mysqlServerGone,
		mysqlServerLost:

		return makeErr(
			"database connection error",
			codeStr,
			errors.Join(dberr.ErrConnection, original),
		)

	default:
		return makeErr(
			"internal database error",
			codeStr,
			errors.Join(dberr.ErrInternal, original),
		)
	}
}

// extractConstraint attempts to extract a constraint/key name from the
// standard MySQL duplicate-key error message.
//
// Example:
//
//	Duplicate entry 'foo' for key 'users.email'
//
// The result is best-effort. MySQL does not expose the constraint as a
// separate field in MySQLError.
func extractConstraint(message string) string {
	const marker = " for key '"

	idx := strings.Index(message, marker)
	if idx == -1 {
		return ""
	}

	value := message[idx+len(marker):]

	if end := strings.IndexByte(value, '\''); end >= 0 {
		return value[:end]
	}

	return ""
}
