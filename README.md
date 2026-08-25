# dberr

**Unified database error handling for Go.**

`dberr` provides a common error model for database errors across different Go database drivers.

Instead of writing database-specific error handling throughout your application:

```go
if errors.Is(err, pgx.ErrNoRows) {
    // ...
}

var pgErr *pgconn.PgError
if errors.As(err, &pgErr) {
    switch pgErr.Code {
    // ...
    }
}
```

you can work with a single application-level API:

```go
err = dberr.Wrap(err)

if errors.Is(err, dberr.ErrNotFound) {
    // resource does not exist
}

if errors.Is(err, dberr.ErrConflict) {
    // unique/exclusion conflict
}

if errors.Is(err, dberr.ErrRetryable) {
    // safe to retry according to application semantics
}
```

At the same time, `dberr` preserves the original driver error, so driver-specific information is not lost.


---

# Supported databases

| Database   | Go driver                        | Status    |
| ---------- | -------------------------------- | --------- |
| PostgreSQL | `github.com/jackc/pgx/v5`        | Supported |
| SQLite     | `modernc.org/sqlite`             | Supported |
| MySQL      | `github.com/go-sql-driver/mysql` | Supported |

The MySQL driver currently supports MySQL 5.7+, MariaDB 10.5+, and is implemented as a native Go `database/sql` driver.

`modernc.org/sqlite` provides a CGo-free SQLite implementation and exposes SQLite result codes through its `Error.Code()` API.


---

## Features

* Unified error categories across database drivers
* Compatible with Go's standard `errors.Is` and `errors.As`
* Preserves original database/driver errors
* Structured information for common database errors
* Driver-specific metadata when it is reliably available
* Transaction error classification
* Retryable error classification
* Context cancellation and timeout handling
* No dependency on a specific database in the core package
* Designed for use in production Go services

Currently supported drivers:

* PostgreSQL / `pgx/v5`
* SQLite / `modernc.org/sqlite`
* MySQL / `go-sql-driver/mysql`

---

## Installation

Install the core package:

```bash
go get github.com/er-davo/dberr
```

Then install the driver adapter you need.

### PostgreSQL

```bash
go get github.com/er-davo/dberr/pgx
```

### SQLite

```bash
go get github.com/er-davo/dberr/sqlite3
```

### MySQL

```bash
go get github.com/er-davo/dberr/mysql
```

---

# Basic usage

Import the core package and the driver adapter.

For PostgreSQL:

```go
import (
    "errors"

    "github.com/er-davo/dberr"
    _ "github.com/er-davo/dberr/pgx"
)
```

For SQLite:

```go
import (
    "errors"

    "github.com/er-davo/dberr"
    _ "github.com/er-davo/dberr/sqlite3"
)
```

For MySQL:

```go
import (
    "errors"

    "github.com/er-davo/dberr"
    _ "github.com/er-davo/dberr/mysql"
)
```

The driver registers itself automatically.

Then wrap database errors at the repository/data-access boundary:

```go
func (r *Repository) GetUser(ctx context.Context, id int64) (*User, error) {
    user, err := r.db.GetUser(ctx, id)
    if err != nil {
        return nil, dberr.Wrap(err)
    }

    return user, nil
}
```

Application code can now work with database-independent errors:

```go
user, err := repo.GetUser(ctx, id)
if err != nil {
    if errors.Is(err, dberr.ErrNotFound) {
        return nil, ErrUserNotFound
    }

    return nil, err
}
```

---

# Error model

The core package defines common database error categories.

## General errors

| Error               | Meaning                                                        |
| ------------------- | -------------------------------------------------------------- |
| `ErrNotFound`       | Requested database resource does not exist                     |
| `ErrConflict`       | Operation conflicts with an existing resource or constraint    |
| `ErrForeignKey`     | Foreign key constraint was violated                            |
| `ErrValidation`     | Database rejected data because of a validation/constraint rule |
| `ErrInternal`       | Internal or otherwise unclassified database error              |
| `ErrNoRowsAffected` | Operation completed but affected no rows                       |
| `ErrForbidden`      | Database operation is forbidden by application/database rules  |

## Infrastructure errors

| Error            | Meaning                                                      |
| ---------------- | ------------------------------------------------------------ |
| `ErrConnection`  | Database connection failure                                  |
| `ErrTransaction` | Transaction-related failure                                  |
| `ErrTimeout`     | Database operation timed out                                 |
| `ErrCanceled`    | Database operation was canceled                              |
| `ErrPermission`  | Database user does not have sufficient privileges            |
| `ErrReadOnly`    | Operation attempted against a read-only database/transaction |
| `ErrRetryable`   | Error may be retried according to application semantics      |

---

# `errors.Is` and `errors.As`

`dberr` is designed around Go's standard error inspection APIs.

For example:

```go
if errors.Is(err, dberr.ErrConflict) {
    // conflict
}
```

You can also retrieve structured information:

```go
var conflict *dberr.ConflictError

if errors.As(err, &conflict) {
    switch conflict.Type {
    case dberr.ConflictUnique:
        // unique constraint
    case dberr.ConflictExclusion:
        // exclusion constraint
    }
}
```

The original database error is preserved as well:

```go
var pgErr *pgconn.PgError

if errors.As(err, &pgErr) {
    fmt.Println(pgErr.Code)
}
```

This allows applications to use the common abstraction while still having access to driver-specific information when necessary.

---

# Structured errors

## ConflictError

`ConflictError` represents a conflict with a database uniqueness/exclusion constraint.

```go
type ConflictError struct {
    Type       ConflictType
    Table      string
    Column     string
    Constraint string
    Err        error
}
```

Conflict types:

```go
const (
    ConflictUnique    ConflictType = "unique"
    ConflictExclusion ConflictType = "exclusion"
)
```

Example:

```go
var conflict *dberr.ConflictError

if errors.As(err, &conflict) {
    fmt.Println(conflict.Type)
    fmt.Println(conflict.Table)
    fmt.Println(conflict.Column)
    fmt.Println(conflict.Constraint)
}
```

Not every database exposes all fields. Empty fields mean that the underlying driver/database does not provide that information reliably.

---

## ForeignKeyError

```go
type ForeignKeyError struct {
    Table           string
    Column          string
    Constraint      string
    ReferencedTable string
    ReferencedColumn string
    Err             error
}
```

Example:

```go
var fkErr *dberr.ForeignKeyError

if errors.As(err, &fkErr) {
    fmt.Println(fkErr.Table)
    fmt.Println(fkErr.Column)
    fmt.Println(fkErr.Constraint)
}
```

PostgreSQL provides the richest foreign-key metadata.

SQLite and MySQL expose less structured information through their Go drivers, so some fields may be empty.

---

## ValidationError

```go
type ValidationError struct {
    Type       ValidationType
    Table      string
    Column     string
    Constraint string
    Err        error
}
```

Validation types:

```go
const (
    ValidationNotNull       ValidationType = "not_null"
    ValidationCheck         ValidationType = "check"
    ValidationTypeMismatch  ValidationType = "type_mismatch"
    ValidationOutOfRange    ValidationType = "out_of_range"
    ValidationInvalidFormat ValidationType = "invalid_format"
)
```

Example:

```go
var validationErr *dberr.ValidationError

if errors.As(err, &validationErr) {
    switch validationErr.Type {
    case dberr.ValidationNotNull:
        // required value is missing

    case dberr.ValidationCheck:
        // CHECK constraint failed

    case dberr.ValidationTypeMismatch:
        // invalid database type

    case dberr.ValidationOutOfRange:
        // value exceeds database limits

    case dberr.ValidationInvalidFormat:
        // invalid representation
    }
}
```

---

# Transaction errors

Transaction failures have their own structured type:

```go
type TransactionError struct {
    Type TransactionType
    Err  error
}
```

Transaction types:

```go
const (
    TransactionClosed        TransactionType = "closed"
    TransactionAborted       TransactionType = "aborted"
    TransactionDeadlock      TransactionType = "deadlock"
    TransactionSerialization TransactionType = "serialization"
)
```

Example:

```go
var txErr *dberr.TransactionError

if errors.As(err, &txErr) {
    switch txErr.Type {
    case dberr.TransactionDeadlock:
        // retry the transaction

    case dberr.TransactionSerialization:
        // retry the transaction

    case dberr.TransactionAborted:
        // transaction must be rolled back

    case dberr.TransactionClosed:
        // transaction is already closed
    }
}
```

Retryable failures can additionally be detected using:

```go
if errors.Is(err, dberr.ErrRetryable) {
    // retry according to application policy
}
```

---

# Driver support

The core API is database-independent, while each driver adapter maps native database errors into the common model.

## PostgreSQL / pgx

Package:

```text
github.com/er-davo/dberr/pgx
```

Underlying driver:

```text
github.com/jackc/pgx/v5
```

PostgreSQL provides particularly rich error metadata through `pgconn.PgError`, including SQLSTATE, table, column and constraint information.

### Supported classifications

| PostgreSQL error               | `dberr` classification | Additional information                                          |
| ------------------------------ | ---------------------- | --------------------------------------------------------------- |
| `23505` UNIQUE                 | `ConflictError`        | type, table, column, constraint                                 |
| `23P01` EXCLUSION              | `ConflictError`        | type, table, column, constraint                                 |
| `23503` FOREIGN KEY            | `ForeignKeyError`      | table, column, constraint, referenced information when provided |
| `23502` NOT NULL               | `ValidationError`      | table, column, constraint                                       |
| `23514` CHECK                  | `ValidationError`      | table, column, constraint                                       |
| `40001` serialization failure  | `TransactionError`     | serialization                                                   |
| `40P01` deadlock               | `TransactionError`     | deadlock                                                        |
| `25006` read-only transaction  | `ErrReadOnly`          | SQLSTATE                                                        |
| `42501` insufficient privilege | `ErrPermission`        | SQLSTATE                                                        |
| `57014` query canceled         | `ErrCanceled`          | SQLSTATE                                                        |
| connection-related `08xxx`     | `ErrConnection`        | SQLSTATE                                                        |
| `pgx.ErrNoRows`                | `ErrNotFound`          | original pgx error                                              |
| `pgx.ErrTxClosed`              | `TransactionError`     | closed                                                          |
| `pgx.ErrTxCommitRollback`      | `TransactionError`     | aborted                                                         |

The original `*pgconn.PgError` is preserved and can be recovered with `errors.As`.

Example:

```go
var pgErr *pgconn.PgError

if errors.As(err, &pgErr) {
    fmt.Println(pgErr.Code)
    fmt.Println(pgErr.TableName)
    fmt.Println(pgErr.ColumnName)
    fmt.Println(pgErr.ConstraintName)
}
```

---

# SQLite / modernc.org/sqlite

Package:

```text
github.com/er-davo/dberr/sqlite3
```

Underlying driver:

```text
modernc.org/sqlite
```

`modernc.org/sqlite` exposes SQLite's result code through `Error.Code()`. This makes it possible to reliably classify SQLite errors using primary and extended result codes.

### Supported classifications

| SQLite error                          | `dberr` classification       | Additional information       |
| ------------------------------------- | ---------------------------- | ---------------------------- |
| `2067` `SQLITE_CONSTRAINT_UNIQUE`     | `ConflictError`              | conflict type                |
| `1555` `SQLITE_CONSTRAINT_PRIMARYKEY` | `ConflictError`              | conflict type                |
| `787` `SQLITE_CONSTRAINT_FOREIGNKEY`  | `ForeignKeyError`            | error code                   |
| `1299` `SQLITE_CONSTRAINT_NOTNULL`    | `ValidationError`            | validation type              |
| `275` `SQLITE_CONSTRAINT_CHECK`       | `ValidationError`            | validation type              |
| `3091` datatype constraint            | `ValidationError`            | type mismatch                |
| `517` `SQLITE_BUSY_SNAPSHOT`          | `TransactionError`           | serialization                |
| `516` `SQLITE_ABORT_ROLLBACK`         | `TransactionError`           | aborted                      |
| `5` `SQLITE_BUSY`                     | `ErrRetryable`               | database busy                |
| `773` `SQLITE_BUSY_TIMEOUT`           | `ErrTimeout`, `ErrRetryable` | lock timeout                 |
| `6` `SQLITE_LOCKED`                   | `ErrRetryable`               | database locked              |
| `8` `SQLITE_READONLY`                 | `ErrReadOnly`                | read-only                    |
| `3` `SQLITE_PERM`                     | `ErrPermission`              | permission denied            |
| `23` `SQLITE_AUTH`                    | `ErrPermission`              | authorization denied         |
| `9` `SQLITE_INTERRUPT`                | `ErrCanceled`                | operation interrupted        |
| `14` `SQLITE_CANTOPEN`                | `ErrConnection`              | database could not be opened |
| `10` `SQLITE_IOERR`                   | `ErrInternal`                | I/O failure                  |
| `11` `SQLITE_CORRUPT`                 | `ErrInternal`                | database corruption          |
| `7` `SQLITE_NOMEM`                    | `ErrInternal`                | memory allocation failure    |
| `18` `SQLITE_TOOBIG`                  | `ValidationError`            | out of range/size            |
| `20` `SQLITE_MISMATCH`                | `ValidationError`            | type mismatch                |

SQLite does not expose the same structured table/column/constraint metadata as PostgreSQL through `modernc.org/sqlite`. Therefore `ConflictError`, `ForeignKeyError` and `ValidationError` may contain only the information that can be reliably derived from the SQLite result code.

The original `*sqlite.Error` remains available:

```go
var sqliteErr *sqlite.Error

if errors.As(err, &sqliteErr) {
    fmt.Println(sqliteErr.Code())
}
```

---

# MySQL

Package:

```text
github.com/er-davo/dberr/mysql
```

Underlying driver:

```text
github.com/go-sql-driver/mysql
```

The Go MySQL driver exposes `MySQLError` with:

* numeric error code
* SQLSTATE
* error message

This information is preserved by `dberr`.

### Supported classifications

| MySQL error                  | `dberr` classification       | Additional information           |
| ---------------------------- | ---------------------------- | -------------------------------- |
| `1062` duplicate entry       | `ConflictError`              | type, best-effort constraint/key |
| `1061` duplicate key name    | `ConflictError`              | type                             |
| `1215` cannot add FK         | `ForeignKeyError`            | error code/message               |
| `1216` no referenced row     | `ForeignKeyError`            | error code/message               |
| `1217` row is referenced     | `ForeignKeyError`            | error code/message               |
| `1451` row is referenced     | `ForeignKeyError`            | error code/message               |
| `1452` no referenced row     | `ForeignKeyError`            | error code/message               |
| `1048` bad NULL              | `ValidationError`            | not-null                         |
| `3819` CHECK violation       | `ValidationError`            | check                            |
| `1213` deadlock              | `TransactionError`           | deadlock, retryable              |
| `1205` lock wait timeout     | `ErrTimeout`, `ErrRetryable` | lock timeout                     |
| read-only transaction errors | `ErrReadOnly`                | SQLSTATE/code                    |
| `1044`/`1045` access denied  | `ErrPermission`              | SQLSTATE/code                    |
| `1142` table access denied   | `ErrPermission`              | SQLSTATE/code                    |
| `1143` column access denied  | `ErrPermission`              | SQLSTATE/code                    |
| `1040` too many connections  | `ErrConnection`              | SQLSTATE/code                    |
| `2006` server gone away      | `ErrConnection`              | SQLSTATE/code                    |
| `2013` server lost           | `ErrConnection`              | SQLSTATE/code                    |

Example:

```go
var mysqlErr *mysql.MySQLError

if errors.As(err, &mysqlErr) {
    fmt.Println(mysqlErr.Number)
    fmt.Println(mysqlErr.SQLState)
    fmt.Println(mysqlErr.Message)
}
```

For duplicate-key errors, `dberr` attempts to extract the key/constraint name from the standard MySQL error message.

For example:

```text
Duplicate entry 'john@example.com' for key 'users.email'
```

may result in:

```go
conflict.Constraint == "users.email"
```

This extraction is intentionally best-effort because the MySQL driver exposes the message rather than separate table/column/constraint fields.

The original `*mysql.MySQLError` is always preserved.

---

# Driver metadata comparison

The three adapters intentionally expose different amounts of structured metadata.

| Information              | PostgreSQL |    MySQL    | SQLite |
| ------------------------ | :--------: | :---------: | :----: |
| Error code               |     Yes    |     Yes     |   Yes  |
| SQLSTATE                 |     Yes    |     Yes     |   No   |
| Original driver error    |     Yes    |     Yes     |   Yes  |
| Conflict type            |     Yes    |     Yes     |   Yes  |
| Table                    |     Yes    |   Limited   |   No   |
| Column                   |     Yes    |   Limited   |   No   |
| Constraint               |     Yes    | Best-effort |   No   |
| Referenced table         |     Yes    |   Limited   |   No   |
| Transaction type         |     Yes    |     Yes     |   Yes  |
| Retryable classification |     Yes    |     Yes     |   Yes  |

The goal is **not** to make every database expose identical metadata. The goal is to provide the strongest common abstraction possible without relying on unreliable parsing or database-specific behavior.

---

# Preserving the original error

`dberr` does not hide the underlying error.

For example:

```go
err := dberr.Wrap(dbErr)
```

The resulting error can be inspected at multiple levels:

```go
errors.Is(err, dberr.ErrConflict)
```

```go
var conflict *dberr.ConflictError
errors.As(err, &conflict)
```

and:

```go
var nativeErr *pgconn.PgError
errors.As(err, &nativeErr)
```

This gives applications both:

1. database-independent error handling;
2. access to database-specific information when necessary.

---

# Recommended application architecture

A good place to call `dberr.Wrap` is the repository/data-access boundary.

```text
HTTP / gRPC
     │
     ▼
Service
     │
     ▼
Repository
     │
     ▼
Database driver
```

Wrap the error when it leaves the database layer:

```go
func (r *Repository) CreateUser(ctx context.Context, user User) error {
    _, err := r.db.ExecContext(
        ctx,
        `INSERT INTO users (email) VALUES (?)`,
        user.Email,
    )

    if err != nil {
        return dberr.Wrap(err)
    }

    return nil
}
```

The service layer does not need to know whether the repository uses PostgreSQL, MySQL or SQLite:

```go
err := repo.CreateUser(ctx, user)
if err != nil {
    if errors.Is(err, dberr.ErrConflict) {
        return ErrEmailAlreadyExists
    }

    return err
}
```

This keeps database-specific logic inside the data-access layer.

---

# Retry handling

`dberr` can classify errors as retryable:

```go
if errors.Is(err, dberr.ErrRetryable) {
    // retry according to application policy
}
```

However, **`ErrRetryable` does not mean "always retry blindly."**

For example, a deadlock usually requires retrying the entire transaction:

```go
for attempt := 0; attempt < maxAttempts; attempt++ {
    err := runTransaction(ctx)

    if err == nil {
        return nil
    }

    if !errors.Is(err, dberr.ErrRetryable) {
        return err
    }

    // backoff
}
```

The retry policy belongs to the application. `dberr` only provides the classification.

---

# Wrapping errors

`dberr.Wrap(nil)` returns `nil`:

```go
if err := dberr.Wrap(err); err != nil {
    // ...
}
```

This makes it safe to use directly at error boundaries.

---

# Design principles

## 1. Use stable database error codes

Driver adapters classify errors using stable native error codes such as PostgreSQL SQLSTATE or SQLite extended result codes rather than parsing human-readable messages.

## 2. Preserve the original error

The native error remains available through Go's `errors.Is` / `errors.As`.

## 3. Do not invent metadata

If a database driver cannot reliably provide a table, column or constraint, the corresponding field remains empty.

## 4. Keep the core database-independent

The core `dberr` package does not depend on PostgreSQL, MySQL or SQLite.

Database-specific dependencies belong to their respective adapters.

## 5. Use standard Go error semantics

The library is built around:

```go
errors.Is(...)
errors.As(...)
errors.Join(...)
```

rather than introducing a custom error inspection API.

---

# Project structure

```text
dberr/
├── dberr.go
├── error.go
├── errors.go
├── conflict.go
├── foreign_key.go
├── transaction.go
├── validation.go
│
├── pgx/
│   ├── pgx.go
│   └── pgx_test.go
│
├── sqlite3/
│   ├── sqlite3.go
│   └── sqlite3_test.go
│
└── mysql/
    ├── mysql.go
    └── mysql_test.go
```

---

# Testing

Run all tests:

```bash
go test ./...
```

The project tests both:

* common `dberr` error semantics;
* driver-specific error mappings.

Driver tests verify that:

* native database errors are classified correctly;
* `errors.Is` works for common error categories;
* `errors.As` retrieves structured errors;
* original driver errors are preserved;
* driver-specific metadata is available when supported.

---

# Adding another database driver

The core package exposes a small driver interface:

```go
type Driver interface {
    Wrap(err error) error
}
```

A new adapter can implement this interface:

```go
type myDriver struct{}

func (d *myDriver) Wrap(err error) error {
    // Map native database errors to dberr errors.
}
```

Then register it:

```go
func init() {
    dberr.Register(&myDriver{})
}
```

A driver adapter should:

1. recognize native database errors;
2. classify them into common `dberr` categories;
3. populate structured metadata when reliably available;
4. preserve the original error;
5. avoid parsing error messages when a stable error code is available.

---

# License

MIT License