package dberr

type Driver interface {
	Wrap(err error) error
}

var driver Driver

func Wrap(err error) error {
	if err == nil {
		return nil
	}
	return driver.Wrap(err)
}

func Register(d Driver) {
	if d == nil {
		panic("dberr: Register driver is nil")
	}
	driver = d
}
