package dberr

type Error struct {
	Msg  string
	Code string
	Err  error
}

func (e *Error) Error() string {
	return e.Msg
}

func (e *Error) Unwrap() error {
	return e.Err
}
