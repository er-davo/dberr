package dberr

type ValidationType string

const (
	ValidationNotNull       ValidationType = "not_null"
	ValidationCheck         ValidationType = "check"
	ValidationTypeMismatch  ValidationType = "type_mismatch"
	ValidationOutOfRange    ValidationType = "out_of_range"
	ValidationInvalidFormat ValidationType = "invalid_format"
)

type ValidationError struct {
	Type       ValidationType
	Table      string
	Column     string
	Constraint string
	Err        error
}

func (e *ValidationError) Error() string {
	return "database validation failed"
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

func (e *ValidationError) Is(target error) bool {
	return target == ErrValidation
}
