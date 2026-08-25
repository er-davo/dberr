package pgx

import (
	"errors"
	"testing"

	"github.com/er-davo/dberr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestWrap_Nil(t *testing.T) {
	if got := (&pgxDriver{}).Wrap(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestWrap_NoRows(t *testing.T) {
	err := (&pgxDriver{}).Wrap(pgx.ErrNoRows)

	if !errors.Is(err, dberr.ErrNotFound) {
		t.Fatal("expected ErrNotFound")
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("expected original pgx.ErrNoRows")
	}
}

func TestWrap_TransactionClosed(t *testing.T) {
	err := (&pgxDriver{}).Wrap(pgx.ErrTxClosed)

	if !errors.Is(err, dberr.ErrTransaction) {
		t.Fatal("expected ErrTransaction")
	}

	if !errors.Is(err, dberr.ErrTransactionClosed) {
		t.Fatal("expected ErrTransactionClosed")
	}

	if !errors.Is(err, pgx.ErrTxClosed) {
		t.Fatal("expected original pgx.ErrTxClosed")
	}

	var txErr *dberr.TransactionError
	if !errors.As(err, &txErr) {
		t.Fatal("expected TransactionError")
	}

	if txErr.Type != dberr.TransactionClosed {
		t.Fatalf(
			"expected transaction type %q, got %q",
			dberr.TransactionClosed,
			txErr.Type,
		)
	}
}

func TestWrap_UniqueViolation(t *testing.T) {
	original := &pgconn.PgError{
		Code:           "23505",
		TableName:      "users",
		ColumnName:     "email",
		ConstraintName: "users_email_key",
		Message:        "duplicate key value violates unique constraint",
	}

	err := (&pgxDriver{}).Wrap(original)

	if !errors.Is(err, dberr.ErrConflict) {
		t.Fatal("expected ErrConflict")
	}

	var conflict *dberr.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatal("expected ConflictError")
	}

	if conflict.Type != dberr.ConflictUnique {
		t.Fatalf("expected unique conflict, got %q", conflict.Type)
	}

	if conflict.Table != "users" {
		t.Fatalf("expected table users, got %q", conflict.Table)
	}

	if conflict.Column != "email" {
		t.Fatalf("expected column email, got %q", conflict.Column)
	}

	if conflict.Constraint != "users_email_key" {
		t.Fatalf("expected constraint users_email_key, got %q", conflict.Constraint)
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatal("expected original *pgconn.PgError")
	}

	if pgErr != original {
		t.Fatal("original PostgreSQL error was not preserved")
	}
}

func TestWrap_ExclusionViolation(t *testing.T) {
	original := &pgconn.PgError{
		Code:           "23P01",
		TableName:      "bookings",
		ConstraintName: "bookings_no_overlap",
	}

	err := (&pgxDriver{}).Wrap(original)

	if !errors.Is(err, dberr.ErrConflict) {
		t.Fatal("expected ErrConflict")
	}

	var conflict *dberr.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatal("expected ConflictError")
	}

	if conflict.Type != dberr.ConflictExclusion {
		t.Fatalf("expected exclusion conflict, got %q", conflict.Type)
	}

	if conflict.Table != "bookings" {
		t.Fatalf("expected table bookings, got %q", conflict.Table)
	}

	if conflict.Constraint != "bookings_no_overlap" {
		t.Fatalf("unexpected constraint: %q", conflict.Constraint)
	}
}

func TestWrap_ForeignKeyViolation(t *testing.T) {
	original := &pgconn.PgError{
		Code:           "23503",
		TableName:      "orders",
		ColumnName:     "user_id",
		ConstraintName: "orders_user_id_fkey",
		Message:        "insert or update violates foreign key constraint",
	}

	err := (&pgxDriver{}).Wrap(original)

	if !errors.Is(err, dberr.ErrForeignKey) {
		t.Fatal("expected ErrForeignKey")
	}

	var fkErr *dberr.ForeignKeyError
	if !errors.As(err, &fkErr) {
		t.Fatal("expected ForeignKeyError")
	}

	if fkErr.Table != "orders" {
		t.Fatalf("unexpected table: %q", fkErr.Table)
	}

	if fkErr.Column != "user_id" {
		t.Fatalf("unexpected column: %q", fkErr.Column)
	}

	if fkErr.Constraint != "orders_user_id_fkey" {
		t.Fatalf("unexpected constraint: %q", fkErr.Constraint)
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatal("expected original PgError")
	}
}

func TestWrap_CheckViolation(t *testing.T) {
	original := &pgconn.PgError{
		Code:           "23514",
		TableName:      "users",
		ColumnName:     "age",
		ConstraintName: "users_age_check",
	}

	err := (&pgxDriver{}).Wrap(original)

	if !errors.Is(err, dberr.ErrValidation) {
		t.Fatal("expected ErrValidation")
	}

	var validationErr *dberr.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatal("expected ValidationError")
	}

	if validationErr.Type != dberr.ValidationCheck {
		t.Fatalf("expected check validation, got %q", validationErr.Type)
	}

	if validationErr.Table != "users" {
		t.Fatalf("unexpected table: %q", validationErr.Table)
	}

	if validationErr.Column != "age" {
		t.Fatalf("unexpected column: %q", validationErr.Column)
	}
}

func TestWrap_NotNullViolation(t *testing.T) {
	original := &pgconn.PgError{
		Code:       "23502",
		TableName:  "users",
		ColumnName: "email",
	}

	err := (&pgxDriver{}).Wrap(original)

	if !errors.Is(err, dberr.ErrValidation) {
		t.Fatal("expected ErrValidation")
	}

	var validationErr *dberr.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatal("expected ValidationError")
	}

	if validationErr.Type != dberr.ValidationNotNull {
		t.Fatalf("expected not_null, got %q", validationErr.Type)
	}
}

func TestWrap_Deadlock(t *testing.T) {
	original := &pgconn.PgError{
		Code: "40P01",
	}

	err := (&pgxDriver{}).Wrap(original)

	if !errors.Is(err, dberr.ErrTransaction) {
		t.Fatal("expected ErrTransaction")
	}

	if !errors.Is(err, dberr.ErrRetryable) {
		t.Fatal("expected ErrRetryable")
	}

	var txErr *dberr.TransactionError
	if !errors.As(err, &txErr) {
		t.Fatal("expected TransactionError")
	}

	if txErr.Type != dberr.TransactionDeadlock {
		t.Fatalf("expected deadlock, got %q", txErr.Type)
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatal("expected original PgError")
	}
}

func TestWrap_SerializationFailure(t *testing.T) {
	original := &pgconn.PgError{
		Code: "40001",
	}

	err := (&pgxDriver{}).Wrap(original)

	if !errors.Is(err, dberr.ErrTransaction) {
		t.Fatal("expected ErrTransaction")
	}

	if !errors.Is(err, dberr.ErrRetryable) {
		t.Fatal("expected ErrRetryable")
	}

	var txErr *dberr.TransactionError
	if !errors.As(err, &txErr) {
		t.Fatal("expected TransactionError")
	}

	if txErr.Type != dberr.TransactionSerialization {
		t.Fatalf("expected serialization, got %q", txErr.Type)
	}
}
