package sqlite3

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/er-davo/dberr"
	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func TestWrap_Nil(t *testing.T) {
	if got := (&sqliteDriver{}).Wrap(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestWrap_UniqueViolation(t *testing.T) {
	db := newTestDB(t)

	_, err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT UNIQUE NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO users (email) VALUES (?)`,
		"test@example.com",
	)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO users (email) VALUES (?)`,
		"test@example.com",
	)
	if err == nil {
		t.Fatal("expected unique constraint error")
	}

	wrapped := (&sqliteDriver{}).Wrap(err)

	if !errors.Is(wrapped, dberr.ErrConflict) {
		t.Fatal("expected ErrConflict")
	}

	var conflict *dberr.ConflictError
	if !errors.As(wrapped, &conflict) {
		t.Fatal("expected ConflictError")
	}

	if conflict.Type != dberr.ConflictUnique {
		t.Fatalf("expected unique, got %q", conflict.Type)
	}
}

func TestWrap_PrimaryKeyViolation(t *testing.T) {
	db := newTestDB(t)

	_, err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO users (id, email) VALUES (?, ?)`,
		1,
		"a@example.com",
	)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO users (id, email) VALUES (?, ?)`,
		1,
		"b@example.com",
	)
	if err == nil {
		t.Fatal("expected primary key violation")
	}

	wrapped := (&sqliteDriver{}).Wrap(err)

	if !errors.Is(wrapped, dberr.ErrConflict) {
		t.Fatal("expected ErrConflict")
	}

	var conflict *dberr.ConflictError
	if !errors.As(wrapped, &conflict) {
		t.Fatal("expected ConflictError")
	}

	if conflict.Type != dberr.ConflictUnique {
		t.Fatalf("expected unique, got %q", conflict.Type)
	}
}

func TestWrap_ForeignKeyViolation(t *testing.T) {
	db := newTestDB(t)

	_, err := db.Exec(`
		PRAGMA foreign_keys = ON;

		CREATE TABLE users (
			id INTEGER PRIMARY KEY
		);

		CREATE TABLE orders (
			id INTEGER PRIMARY KEY,
			user_id INTEGER,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
	`)
	if err != nil {
		t.Fatalf("create tables: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO orders (user_id) VALUES (?)`,
		999,
	)
	if err == nil {
		t.Fatal("expected foreign key violation")
	}

	wrapped := (&sqliteDriver{}).Wrap(err)

	if !errors.Is(wrapped, dberr.ErrForeignKey) {
		t.Fatal("expected ErrForeignKey")
	}

	var fkErr *dberr.ForeignKeyError
	if !errors.As(wrapped, &fkErr) {
		t.Fatal("expected ForeignKeyError")
	}
}

func TestWrap_NotNullViolation(t *testing.T) {
	db := newTestDB(t)

	_, err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO users (email) VALUES (NULL)`,
	)
	if err == nil {
		t.Fatal("expected NOT NULL violation")
	}

	wrapped := (&sqliteDriver{}).Wrap(err)

	if !errors.Is(wrapped, dberr.ErrValidation) {
		t.Fatal("expected ErrValidation")
	}

	var validationErr *dberr.ValidationError
	if !errors.As(wrapped, &validationErr) {
		t.Fatal("expected ValidationError")
	}

	if validationErr.Type != dberr.ValidationNotNull {
		t.Fatalf(
			"expected not_null, got %q",
			validationErr.Type,
		)
	}
}

func TestWrap_CheckViolation(t *testing.T) {
	db := newTestDB(t)

	_, err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			age INTEGER CHECK (age >= 18)
		)
	`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO users (age) VALUES (?)`,
		10,
	)
	if err == nil {
		t.Fatal("expected CHECK violation")
	}

	wrapped := (&sqliteDriver{}).Wrap(err)

	if !errors.Is(wrapped, dberr.ErrValidation) {
		t.Fatal("expected ErrValidation")
	}

	var validationErr *dberr.ValidationError
	if !errors.As(wrapped, &validationErr) {
		t.Fatal("expected ValidationError")
	}

	if validationErr.Type != dberr.ValidationCheck {
		t.Fatalf(
			"expected check, got %q",
			validationErr.Type,
		)
	}
}

func TestWrap_ContextCanceled(t *testing.T) {
	err := (&sqliteDriver{}).Wrap(context.Canceled)

	if !errors.Is(err, context.Canceled) {
		t.Fatal("expected original context.Canceled")
	}
}
