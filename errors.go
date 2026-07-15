package dberr

var (
	ErrNotFound       = &Error{}
	ErrConflict       = &Error{}
	ErrForeignKey     = &Error{}
	ErrValidation     = &Error{}
	ErrNoRowsAffected = &Error{}
	ErrForbidden      = &Error{}

	ErrInternal = &Error{}
)
