package dberr

type ConflictType string

const (
	ConflictUnique    ConflictType = "unique"
	ConflictExclusion ConflictType = "exclusion"
)

type ConflictError struct {
	Type       ConflictType
	Table      string
	Column     string
	Constraint string
	Err        error
}

func (e *ConflictError) Error() string {
	return "database conflict"
}

func (e *ConflictError) Unwrap() error {
	return e.Err
}

func (e *ConflictError) Is(target error) bool {
	return target == ErrConflict
}
