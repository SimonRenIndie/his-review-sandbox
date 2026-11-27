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

func columns(t *testing.T, d *sql.DB) []string {
	t.Helper()
	rows, err := d.Query(`SELECT name FROM pragma_table_info('appointments') ORDER BY cid`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatal(err)
		}
		cols = append(cols, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return cols
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
	var withCode, withNotes int
	if err := d.QueryRow(`SELECT
		COUNT(*) FILTER (WHERE cancel_reason_code <> ''),
		COUNT(*) FILTER (WHERE cancel_notes IS NOT NULL)
		FROM appointments`).Scan(&withCode, &withNotes); err != nil {
		t.Fatal(err)
	}
	if withCode != 0 || withNotes != 0 {
		t.Fatalf("existing rows got %d codes and %d notes, want none", withCode, withNotes)
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
