package dberr_test

import (
	"errors"
	"testing"

	"github.com/er-davo/dberr"
)

func TestTransactionError(t *testing.T) {
	tests := []struct {
		name          string
		transaction   dberr.TransactionType
		expectedError error
	}{
		{
			name:          "closed",
			transaction:   dberr.TransactionClosed,
			expectedError: dberr.ErrTransactionClosed,
		},
		{
			name:          "aborted",
			transaction:   dberr.TransactionAborted,
			expectedError: dberr.ErrTransactionAborted,
		},
		{
			name:          "deadlock",
			transaction:   dberr.TransactionDeadlock,
			expectedError: nil,
		},
		{
			name:          "serialization",
			transaction:   dberr.TransactionSerialization,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := errors.New("transaction error")

			err := &dberr.TransactionError{
				Type: tt.transaction,
				Err:  original,
			}

			if !errors.Is(err, dberr.ErrTransaction) {
				t.Fatal("expected ErrTransaction")
			}

			if !errors.Is(err, original) {
				t.Fatal("expected original error to be preserved")
			}

			var txErr *dberr.TransactionError
			if !errors.As(err, &txErr) {
				t.Fatal("expected TransactionError")
			}

			if txErr.Type != tt.transaction {
				t.Fatalf("expected transaction type %q, got %q",
					tt.transaction, txErr.Type)
			}

			if tt.expectedError != nil {
				if !errors.Is(err, tt.expectedError) {
					t.Fatalf("expected %v", tt.expectedError)
				}
			}
		})
	}
}

func TestTransactionErrorDoesNotMatchWrongType(t *testing.T) {
	err := &dberr.TransactionError{
		Type: dberr.TransactionClosed,
	}

	if errors.Is(err, dberr.ErrTransactionAborted) {
		t.Fatal("closed transaction must not match ErrTransactionAborted")
	}
}
