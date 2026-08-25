package dberr_test

import (
	"errors"
	"testing"

	"github.com/er-davo/dberr"
)

func TestErrorWithJoinedErrors(t *testing.T) {
	original := errors.New("original database error")

	err := &dberr.Error{
		Msg:  "database conflict",
		Code: "23505",
		Err: errors.Join(
			dberr.ErrConflict,
			original,
		),
	}

	if !errors.Is(err, dberr.ErrConflict) {
		t.Fatal("expected ErrConflict")
	}

	if !errors.Is(err, original) {
		t.Fatal("expected original error")
	}
}
