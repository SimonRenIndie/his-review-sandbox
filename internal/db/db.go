// Package db opens the SQLite database and applies migrations in order.
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

//go:embed seed.sql
var seed string

// Open opens the database file at path.
func Open(path string) (*sql.DB, error) { return sql.Open("sqlite", path) }

// Up applies every .up.sql in name order.
func Up(db *sql.DB) error { return apply(db, ".up.sql", false) }

// Down applies every .down.sql in reverse name order.
func Down(db *sql.DB) error { return apply(db, ".down.sql", true) }

func apply(db *sql.DB, suffix string, reverse bool) error {
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), suffix) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if reverse {
		sort.Sort(sort.Reverse(sort.StringSlice(names)))
	}
	for _, n := range names {
		b, err := migrations.ReadFile("migrations/" + n)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(b)); err != nil {
			return fmt.Errorf("%s: %w", n, err)
		}
	}
	return nil
}

// Seed loads seed.sql.
func Seed(db *sql.DB) error { _, err := db.Exec(seed); return err }

// OpenTest returns a migrated, seeded database in a temp dir.
func OpenTest(t *testing.T) *sql.DB {
	t.Helper()
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if err := Up(d); err != nil {
		t.Fatal(err)
	}
	if err := Seed(d); err != nil {
		t.Fatal(err)
	}
	return d
}
