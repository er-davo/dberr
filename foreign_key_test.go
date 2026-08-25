package dberr_test

import (
	"errors"
	"testing"

	"github.com/er-davo/dberr"
)

func TestForeignKeyError(t *testing.T) {
	original := errors.New("foreign key violation")

	err := &dberr.ForeignKeyError{
		Table:            "orders",
		Column:           "user_id",
		Constraint:       "orders_user_id_fkey",
		ReferencedTable:  "users",
		ReferencedColumn: "id",
		Err:              original,
	}

	if !errors.Is(err, dberr.ErrForeignKey) {
		t.Fatal("expected ErrForeignKey")
	}

	if !errors.Is(err, original) {
		t.Fatal("expected original error to be preserved")
	}

	var fkErr *dberr.ForeignKeyError
	if !errors.As(err, &fkErr) {
		t.Fatal("expected ForeignKeyError")
	}

	if fkErr.Table != "orders" {
		t.Fatalf("expected table %q, got %q",
			"orders", fkErr.Table)
	}

	if fkErr.Column != "user_id" {
		t.Fatalf("expected column %q, got %q",
			"user_id", fkErr.Column)
	}

	if fkErr.Constraint != "orders_user_id_fkey" {
		t.Fatalf("expected constraint %q, got %q",
			"orders_user_id_fkey", fkErr.Constraint)
	}

	if fkErr.ReferencedTable != "users" {
		t.Fatalf("expected referenced table %q, got %q",
			"users", fkErr.ReferencedTable)
	}

	if fkErr.ReferencedColumn != "id" {
		t.Fatalf("expected referenced column %q, got %q",
			"id", fkErr.ReferencedColumn)
	}
}
