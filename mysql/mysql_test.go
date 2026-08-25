package mysql

import (
	"errors"
	"testing"

	"github.com/er-davo/dberr"
	gomysql "github.com/go-sql-driver/mysql"
)

func TestWrap_Nil(t *testing.T) {
	if got := (&mysqlDriver{}).Wrap(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestWrap_UniqueViolation(t *testing.T) {
	original := &gomysql.MySQLError{
		Number:   1062,
		SQLState: [5]byte{'2', '3', '0', '0', '0'},
		Message:  "Duplicate entry 'test@example.com' for key 'users.email'",
	}

	err := (&mysqlDriver{}).Wrap(original)

	if !errors.Is(err, dberr.ErrConflict) {
		t.Fatal("expected ErrConflict")
	}

	var conflict *dberr.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatal("expected ConflictError")
	}

	if conflict.Type != dberr.ConflictUnique {
		t.Fatalf("expected unique, got %q", conflict.Type)
	}

	if conflict.Constraint != "users.email" {
		t.Fatalf(
			"expected constraint users.email, got %q",
			conflict.Constraint,
		)
	}

	var mysqlErr *gomysql.MySQLError
	if !errors.As(err, &mysqlErr) {
		t.Fatal("expected original MySQLError")
	}

	if mysqlErr != original {
		t.Fatal("original MySQL error was not preserved")
	}
}

func TestWrap_DuplicateKeyName(t *testing.T) {
	original := &gomysql.MySQLError{
		Number:  1061,
		Message: "Duplicate key name 'users_email_key'",
	}

	err := (&mysqlDriver{}).Wrap(original)

	if !errors.Is(err, dberr.ErrConflict) {
		t.Fatal("expected ErrConflict")
	}

	var conflict *dberr.ConflictError
	if !errors.As(err, &conflict) {
		t.Fatal("expected ConflictError")
	}

	if conflict.Type != dberr.ConflictUnique {
		t.Fatalf("expected unique, got %q", conflict.Type)
	}
}

func TestWrap_ForeignKeyViolation(t *testing.T) {
	tests := []uint16{
		1215,
		1216,
		1217,
		1451,
		1452,
	}

	for _, code := range tests {
		t.Run(string(rune(code)), func(t *testing.T) {
			original := &gomysql.MySQLError{
				Number:  code,
				Message: "foreign key violation",
			}

			err := (&mysqlDriver{}).Wrap(original)

			if !errors.Is(err, dberr.ErrForeignKey) {
				t.Fatal("expected ErrForeignKey")
			}

			var fkErr *dberr.ForeignKeyError
			if !errors.As(err, &fkErr) {
				t.Fatal("expected ForeignKeyError")
			}

			var mysqlErr *gomysql.MySQLError
			if !errors.As(err, &mysqlErr) {
				t.Fatal("expected original MySQLError")
			}
		})
	}
}

func TestWrap_NotNullViolation(t *testing.T) {
	original := &gomysql.MySQLError{
		Number:  1048,
		Message: "Column 'email' cannot be null",
	}

	err := (&mysqlDriver{}).Wrap(original)

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

func TestWrap_CheckViolation(t *testing.T) {
	original := &gomysql.MySQLError{
		Number:  3819,
		Message: "Check constraint 'users_age_check' is violated",
	}

	err := (&mysqlDriver{}).Wrap(original)

	if !errors.Is(err, dberr.ErrValidation) {
		t.Fatal("expected ErrValidation")
	}

	var validationErr *dberr.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatal("expected ValidationError")
	}

	if validationErr.Type != dberr.ValidationCheck {
		t.Fatalf("expected check, got %q", validationErr.Type)
	}
}

func TestWrap_Deadlock(t *testing.T) {
	original := &gomysql.MySQLError{
		Number:  1213,
		Message: "Deadlock found when trying to get lock",
	}

	err := (&mysqlDriver{}).Wrap(original)

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
}

func TestWrap_LockWaitTimeout(t *testing.T) {
	original := &gomysql.MySQLError{
		Number:  1205,
		Message: "Lock wait timeout exceeded",
	}

	err := (&mysqlDriver{}).Wrap(original)

	if !errors.Is(err, dberr.ErrTimeout) {
		t.Fatal("expected ErrTimeout")
	}

	if !errors.Is(err, dberr.ErrRetryable) {
		t.Fatal("expected ErrRetryable")
	}

	if errors.Is(err, dberr.ErrTransaction) {
		t.Fatal("lock wait timeout should not be classified as transaction error")
	}
}

func TestWrap_PermissionDenied(t *testing.T) {
	tests := []uint16{
		1044,
		1045,
		1142,
		1143,
	}

	for _, code := range tests {
		t.Run(string(rune(code)), func(t *testing.T) {
			err := (&mysqlDriver{}).Wrap(&gomysql.MySQLError{
				Number: code,
			})

			if !errors.Is(err, dberr.ErrPermission) {
				t.Fatal("expected ErrPermission")
			}
		})
	}
}

func TestWrap_ReadOnly(t *testing.T) {
	for _, code := range []uint16{1207, 1290} {
		t.Run(string(rune(code)), func(t *testing.T) {
			err := (&mysqlDriver{}).Wrap(&gomysql.MySQLError{
				Number: code,
			})

			if !errors.Is(err, dberr.ErrReadOnly) {
				t.Fatal("expected ErrReadOnly")
			}
		})
	}
}

func TestWrap_ConnectionError(t *testing.T) {
	tests := []uint16{
		1040,
		2006,
		2013,
	}

	for _, code := range tests {
		t.Run(string(rune(code)), func(t *testing.T) {
			err := (&mysqlDriver{}).Wrap(&gomysql.MySQLError{
				Number: code,
			})

			if !errors.Is(err, dberr.ErrConnection) {
				t.Fatal("expected ErrConnection")
			}
		})
	}
}

func TestExtractConstraint(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "standard duplicate entry",
			message: "Duplicate entry 'foo' for key 'users.email'",
			want:    "users.email",
		},
		{
			name:    "no constraint",
			message: "Duplicate entry 'foo'",
			want:    "",
		},
		{
			name:    "empty",
			message: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractConstraint(tt.message)

			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
