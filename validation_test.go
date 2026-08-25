package dberr_test

import (
	"errors"
	"testing"

	"github.com/er-davo/dberr"
)

func TestValidationError(t *testing.T) {
	tests := []struct {
		name string
		typ  dberr.ValidationType
	}{
		{
			name: "not null",
			typ:  dberr.ValidationNotNull,
		},
		{
			name: "check",
			typ:  dberr.ValidationCheck,
		},
		{
			name: "type mismatch",
			typ:  dberr.ValidationTypeMismatch,
		},
		{
			name: "out of range",
			typ:  dberr.ValidationOutOfRange,
		},
		{
			name: "invalid format",
			typ:  dberr.ValidationInvalidFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := errors.New("validation error")

			err := &dberr.ValidationError{
				Type:       tt.typ,
				Table:      "users",
				Column:     "age",
				Constraint: "users_age_check",
				Err:        original,
			}

			if !errors.Is(err, dberr.ErrValidation) {
				t.Fatal("expected ErrValidation")
			}

			if !errors.Is(err, original) {
				t.Fatal("expected original error to be preserved")
			}

			var validationErr *dberr.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatal("expected ValidationError")
			}

			if validationErr.Type != tt.typ {
				t.Fatalf("expected validation type %q, got %q",
					tt.typ, validationErr.Type)
			}

			if validationErr.Table != "users" {
				t.Fatalf("expected table %q, got %q",
					"users", validationErr.Table)
			}

			if validationErr.Column != "age" {
				t.Fatalf("expected column %q, got %q",
					"age", validationErr.Column)
			}

			if validationErr.Constraint != "users_age_check" {
				t.Fatalf("expected constraint %q, got %q",
					"users_age_check", validationErr.Constraint)
			}
		})
	}
}
