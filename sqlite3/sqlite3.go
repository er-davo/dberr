package sqlite3

import (
	"errors"
	"fmt"

	"github.com/er-davo/dberr"
	"modernc.org/sqlite"
)

func init() {
	dberr.Register(&sqliteDriver{})
}

func makeErr(msg string, code string, err error, srcErr error) error {
	if err == nil {
		return &dberr.Error{
			Msg:  msg,
			Code: code,
			Err:  srcErr,
		}
	}
	return &dberr.Error{
		Msg:  msg,
		Code: code,
		Err:  fmt.Errorf("%w: %w", err, srcErr),
	}
}

type sqliteDriver struct{}

func (s *sqliteDriver) Wrap(err error) error {
	if err == nil {
		return nil
	}

	var sqlErr *sqlite.Error
	if errors.As(err, &sqlErr) {
		// В modernc код ошибки получается методом Code()
		code := sqlErr.Code()
		codeStr := fmt.Sprintf("%d", code)

		// Маппим по расширенным кодам (Extended Error Codes) SQLite
		switch code {
		// 2067 = SQLITE_CONSTRAINT_UNIQUE, 1555 = SQLITE_CONSTRAINT_PRIMARYKEY
		case 2067, 1555:
			return makeErr(
				"record not unique",
				codeStr,
				dberr.ErrConflict,
				err,
			)
		// 787 = SQLITE_CONSTRAINT_FOREIGNKEY
		case 787:
			return makeErr(
				"foreign key violation",
				codeStr,
				dberr.ErrForeignKey,
				err,
			)
		// 1293 = SQLITE_CONSTRAINT_NOTNULL, 275 = SQLITE_CONSTRAINT_CHECK
		case 1293, 275:
			return makeErr(
				"check violation",
				codeStr,
				dberr.ErrValidation,
				err,
			)
		default:
			// Проверяем базовый код (Primary Result Code) через остаток от деления на 256
			// 19 = SQLITE_CONSTRAINT
			if code%256 == 19 {
				return makeErr(
					"database constraint failed",
					codeStr,
					dberr.ErrValidation,
					err,
				)
			}

			return makeErr(
				"internal error",
				codeStr,
				dberr.ErrInternal,
				err,
			)
		}
	}

	return makeErr(
		"internal error",
		"internal",
		dberr.ErrInternal,
		err,
	)
}
