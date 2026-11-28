package db

import (
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"
)

func execFile(t *testing.T, d *sql.DB, name string) {
	t.Helper()
	b, err := migrations.ReadFile("migrations/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(string(b)); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
}

type column struct {
	Name, Type string
	NotNull    bool
	Default    sql.NullString
	PK         int
}

// columns returns the full definition of every appointments column, in order.
func columns(t *testing.T, d *sql.DB) []column {
	t.Helper()
	rows, err := d.Query(`SELECT name, type, "notnull", dflt_value, pk FROM pragma_table_info('appointments') ORDER BY cid`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var cols []column
	for rows.Next() {
		var c column
		if err := rows.Scan(&c.Name, &c.Type, &c.NotNull, &c.Default, &c.PK); err != nil {
			t.Fatal(err)
		}
		cols = append(cols, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return cols
}

// newColumns returns cancel_reason_code and cancel_notes for every row, by id.
// A NULL code fails the scan.
func newColumns(t *testing.T, d *sql.DB) map[string]sql.NullString {
	t.Helper()
	rows, err := d.Query(`SELECT id, cancel_reason_code, cancel_notes FROM appointments ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	notes := map[string]sql.NullString{}
	for rows.Next() {
		var id, code string
		var n sql.NullString
		if err := rows.Scan(&id, &code, &n); err != nil {
			t.Fatalf("scan new columns: %v", err)
		}
		if code != "" {
			t.Fatalf("%s: cancel_reason_code %q, want empty", id, code)
		}
		notes[id] = n
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return notes
}

type baseRow struct{ ID, Facility, Patient, Status, Reason string }

func baseRows(t *testing.T, d *sql.DB) []baseRow {
	t.Helper()
	rows, err := d.Query(`SELECT id, facility_id, patient_name, status, cancel_reason FROM appointments ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []baseRow
	for rows.Next() {
		var r baseRow
		if err := rows.Scan(&r.ID, &r.Facility, &r.Patient, &r.Status, &r.Reason); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// Rows that exist before 002 runs keep every value, get an empty code and
// NULL notes; 002 down returns the table to 001's shape with the same values,
// and 002 can be applied again afterwards.
func TestAC4_P2_Migration002KeepsRows(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "migrate.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })

	execFile(t, d, "001_appointments.up.sql")
	if err := Seed(d); err != nil {
		t.Fatal(err)
	}
	mainCols := columns(t, d)
	before := baseRows(t, d)
	if len(before) != 4 || before[1].Reason != "patient called, travelling" || before[3].Reason != "clinic closed for maintenance" {
		t.Fatalf("unexpected seed: %+v", before)
	}

	execFile(t, d, "002_cancel_reason_code.up.sql")
	if got := baseRows(t, d); !reflect.DeepEqual(got, before) {
		t.Fatalf("up changed existing rows:\n got %+v\nwant %+v", got, before)
	}
	notes := newColumns(t, d)
	if len(notes) != len(before) {
		t.Fatalf("read %d rows after up, want %d", len(notes), len(before))
	}
	for id, n := range notes {
		if n.Valid {
			t.Fatalf("%s: cancel_notes %q, want NULL", id, n.String)
		}
	}

	execFile(t, d, "002_cancel_reason_code.down.sql")
	if got := columns(t, d); !reflect.DeepEqual(got, mainCols) {
		t.Fatalf("down left columns %v, want %v", got, mainCols)
	}
	if got := baseRows(t, d); !reflect.DeepEqual(got, before) {
		t.Fatalf("down changed rows:\n got %+v\nwant %+v", got, before)
	}

	execFile(t, d, "002_cancel_reason_code.up.sql")
	if got := baseRows(t, d); !reflect.DeepEqual(got, before) {
		t.Fatalf("second up changed rows:\n got %+v\nwant %+v", got, before)
	}
}
