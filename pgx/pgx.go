package pgx

import (
	"errors"
	"fmt"

	"github.com/er-davo/dberr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func init() {
	dberr.Register(&pgxDriver{})
}

func makeErr(
	msg string,
	code string,
	err error,
	pgErr error,
) error {
	if err == nil {
		return &dberr.Error{
			Msg:  msg,
			Code: code,
			Err:  pgErr,
		}
	}

	err1 := fmt.Errorf("%w: %w", err, pgErr)

	return &dberr.Error{
		Msg:  msg,
		Code: code,
		Err:  err1,
	}
}

// Postgres error codes constants
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgCheckViolation      = "23514"
	pgNotNullViolation    = "23502"
	pgExclusionViolation  = "23P01"
)

type pgxDriver struct{}

// TODO: add more errors (in dberr package)
func (p *pgxDriver) Wrap(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return makeErr(
			"record not found",
			"not_found",
			dberr.ErrNotFound,
			err,
		)
	}

	if errors.Is(err, pgx.ErrTxClosed) {
		return makeErr(
			"transaction closed",
			"tx_closed",
			nil,
			err,
		)
	}

	if errors.Is(err, pgx.ErrTxCommitRollback) {
		return makeErr(
			"transaction commit rollback",
			"tx_commit_rollback",
			nil,
			err,
		)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		code := pgErr.Code
		switch code {
		case pgUniqueViolation, pgExclusionViolation:
			return makeErr(
				"record not unique",
				code,
				dberr.ErrConflict,
				err,
			)
		case pgForeignKeyViolation:
			return makeErr(
				"foreign key violation",
				code,
				dberr.ErrForeignKey,
				err,
			)
		case pgCheckViolation, pgNotNullViolation:
			return makeErr(
				"check violation",
				code,
				dberr.ErrValidation,
				err,
			)
		default:
			// Unhandled postgres error
			return makeErr(
				"internal error",
				code,
				dberr.ErrInternal,
				err,
			)
		}
	}

	// Fallback for other errors (connection, etc.)
	return makeErr(
		"internal error",
		"internal",
		dberr.ErrInternal,
		err,
	)
}
