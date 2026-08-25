package dberr_test

import (
	"errors"
	"testing"

	"github.com/er-davo/dberr"
)

func TestConflictError(t *testing.T) {
	t.Run("unique conflict", func(t *testing.T) {
		original := errors.New("duplicate key")

		err := &dberr.ConflictError{
			Type:       dberr.ConflictUnique,
			Table:      "users",
			Column:     "email",
			Constraint: "users_email_key",
			Err:        original,
		}

		if !errors.Is(err, dberr.ErrConflict) {
			t.Fatal("expected ErrConflict")
		}

		if !errors.Is(err, original) {
			t.Fatal("expected original error to be preserved")
		}

		var conflict *dberr.ConflictError
		if !errors.As(err, &conflict) {
			t.Fatal("expected ConflictError")
		}

		if conflict.Type != dberr.ConflictUnique {
			t.Fatalf("expected conflict type %q, got %q",
				dberr.ConflictUnique, conflict.Type)
		}

		if conflict.Table != "users" {
			t.Fatalf("expected table %q, got %q",
				"users", conflict.Table)
		}

		if conflict.Column != "email" {
			t.Fatalf("expected column %q, got %q",
				"email", conflict.Column)
		}

		if conflict.Constraint != "users_email_key" {
			t.Fatalf("expected constraint %q, got %q",
				"users_email_key", conflict.Constraint)
		}
	})

	t.Run("exclusion conflict", func(t *testing.T) {
		err := &dberr.ConflictError{
			Type:       dberr.ConflictExclusion,
			Table:      "bookings",
			Constraint: "bookings_no_overlap",
		}

		if !errors.Is(err, dberr.ErrConflict) {
			t.Fatal("expected ErrConflict")
		}

		var conflict *dberr.ConflictError
		if !errors.As(err, &conflict) {
			t.Fatal("expected ConflictError")
		}

		if conflict.Type != dberr.ConflictExclusion {
			t.Fatalf("expected conflict type %q, got %q",
				dberr.ConflictExclusion, conflict.Type)
		}
	})
}
