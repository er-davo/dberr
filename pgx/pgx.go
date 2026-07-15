package pgx

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type pgxDriver struct{}

func f() {
	dberr.Register(&pgxDriver{})
}

// Postgres error codes constants
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgCheckViolation      = "23514"
	pgNotNullViolation    = "23502"
	pgExclusionViolation  = "23P01"
)

func (p *pgxDriver) Wrap(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation, pgExclusionViolation:
			return fmt.Errorf("%w: %s", ErrConflict, pgErr.Detail)
		case pgForeignKeyViolation:
			return fmt.Errorf("%w: %s", ErrForeignKey, pgErr.Detail)
		case pgCheckViolation, pgNotNullViolation:
			return fmt.Errorf("%w: %s", ErrValidation, pgErr.Message)
		default:
			// Unhandled postgres error
			return fmt.Errorf("%w (pg code %s): %v", ErrInternal, pgErr.Code, pgErr.Message)
		}
	}

	// Fallback for other errors (connection, etc.)
	return fmt.Errorf("%w: %v", ErrInternal, err)
}
