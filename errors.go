package dberr

import "errors"

var (
	ErrNotFound       = errors.New("resource not found")
	ErrConflict       = errors.New("resource already exists or conflicts")
	ErrForeignKey     = errors.New("referenced resource does not exist")
	ErrValidation     = errors.New("database validation failed")
	ErrInternal       = errors.New("internal database error")
	ErrNoRowsAffected = errors.New("no rows affected")
	ErrForbidden      = errors.New("this action is forbidden")
)
