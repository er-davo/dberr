package dberr

type ForeignKeyError struct {
	Table            string
	Column           string
	Constraint       string
	ReferencedTable  string
	ReferencedColumn string
	Err              error
}

func (e *ForeignKeyError) Error() string {
	return "foreign key violation"
}

func (e *ForeignKeyError) Unwrap() error {
	return e.Err
}

func (e *ForeignKeyError) Is(target error) bool {
	return target == ErrForeignKey
}
