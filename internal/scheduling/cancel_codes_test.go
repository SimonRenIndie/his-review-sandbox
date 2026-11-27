package scheduling

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"

	"github.com/SimonRenIndie/his-review-sandbox/internal/db"
)

// cancelRow is every column a cancel can change.
type cancelRow struct {
	Status, Reason, Code string
	Notes                sql.NullString
}

func readCancelRow(t *testing.T, d *sql.DB, id string) cancelRow {
	t.Helper()
	var r cancelRow
	err := d.QueryRow(`SELECT status, cancel_reason, cancel_reason_code, cancel_notes FROM appointments WHERE id = ?`, id).
		Scan(&r.Status, &r.Reason, &r.Code, &r.Notes)
	if err != nil {
		t.Fatalf("read %s: %v", id, err)
	}
	return r
}

func TestAC1_UnknownCodeRefused(t *testing.T) {
	for _, code := range []string{"bored", "PATIENT_REQUEST", " other", "other ", ""} {
		t.Run(code, func(t *testing.T) {
			s := &Service{DB: db.OpenTest(t)}
			before := readCancelRow(t, s.DB, "a1")
			err := s.CancelWithCode(context.Background(), Caller{"u1", "f1"}, "a1", code, "a note long enough")
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("want ErrInvalid, got %v", err)
			}
			if got := readCancelRow(t, s.DB, "a1"); got != before {
				t.Fatalf("a1 changed: %+v -> %+v", before, got)
			}
		})
	}
}

func TestAC1_EveryListedCodeAccepted(t *testing.T) {
	for _, code := range []string{"patient_request", "provider_unavailable", "clinic_closure", "booked_in_error", "other"} {
		t.Run(code, func(t *testing.T) {
			s := &Service{DB: db.OpenTest(t)}
			if err := s.CancelWithCode(context.Background(), Caller{"u1", "f1"}, "a1", code, "a note long enough"); err != nil {
				t.Fatal(err)
			}
			got := readCancelRow(t, s.DB, "a1")
			if got.Status != "cancelled" || got.Code != code {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

func TestAC2_OtherNeedsNote(t *testing.T) {
	refused := map[string]string{
		"missing":              "",
		"short":                "short",
		"9 runes after trim":   "   abcdefghi   ",
		"only spaces":          "             ",
		"10 bytes but 5 runes": "مرحبا",
	}
	for name, note := range refused {
		t.Run(name, func(t *testing.T) {
			s := &Service{DB: db.OpenTest(t)}
			before := readCancelRow(t, s.DB, "a1")
			err := s.CancelWithCode(context.Background(), Caller{"u1", "f1"}, "a1", "other", note)
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("note %q: want ErrInvalid, got %v", note, err)
			}
			if got := readCancelRow(t, s.DB, "a1"); got != before {
				t.Fatalf("a1 changed: %+v -> %+v", before, got)
			}
		})
	}

	t.Run("10 runes after trim accepted", func(t *testing.T) {
		s := &Service{DB: db.OpenTest(t)}
		if err := s.CancelWithCode(context.Background(), Caller{"u1", "f1"}, "a1", "other", "  abcdefghij  "); err != nil {
			t.Fatal(err)
		}
		got := readCancelRow(t, s.DB, "a1")
		want := cancelRow{Status: "cancelled", Reason: "Other", Code: "other", Notes: sql.NullString{String: "abcdefghij", Valid: true}}
		if got != want {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})
}

func TestAC3_StoresCodeLabelAndNote(t *testing.T) {
	s := &Service{DB: db.OpenTest(t)}
	if err := s.CancelWithCode(context.Background(), Caller{"u1", "f1"}, "a1", "clinic_closure", " power cut "); err != nil {
		t.Fatal(err)
	}
	got := readCancelRow(t, s.DB, "a1")
	want := cancelRow{Status: "cancelled", Reason: "Clinic or service closed", Code: "clinic_closure", Notes: sql.NullString{String: "power cut", Valid: true}}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestAC3_BlankNoteStoredAsNull(t *testing.T) {
	for name, note := range map[string]string{"empty": "", "spaces": "   ", "whitespace": " \t\n "} {
		t.Run(name, func(t *testing.T) {
			s := &Service{DB: db.OpenTest(t)}
			if err := s.CancelWithCode(context.Background(), Caller{"u1", "f1"}, "a1", "booked_in_error", note); err != nil {
				t.Fatal(err)
			}
			var isNull bool
			if err := s.DB.QueryRow(`SELECT cancel_notes IS NULL FROM appointments WHERE id = 'a1'`).Scan(&isNull); err != nil {
				t.Fatal(err)
			}
			if !isNull {
				t.Fatalf("note %q: cancel_notes is not NULL", note)
			}
			got := readCancelRow(t, s.DB, "a1")
			if got.Status != "cancelled" || got.Code != "booked_in_error" || got.Reason != "Booked in error" {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

func TestAC5_P1_OtherFacilityRefused(t *testing.T) {
	cases := []struct {
		name   string
		caller Caller
		id     string
	}{
		{"booked at another facility", Caller{"u1", "f1"}, "a3"},
		{"cancelled at another facility", Caller{"u1", "f1"}, "a4"},
		{"caller without facility", Caller{"", ""}, "a1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Service{DB: db.OpenTest(t)}
			before := readCancelRow(t, s.DB, tc.id)
			err := s.CancelWithCode(context.Background(), tc.caller, tc.id, "patient_request", "")
			if !errors.Is(err, ErrForbidden) {
				t.Fatalf("%s: want ErrForbidden, got %v", tc.id, err)
			}
			if got := readCancelRow(t, s.DB, tc.id); got != before {
				t.Fatalf("%s changed: %+v -> %+v", tc.id, before, got)
			}
		})
	}
}

func TestAC5_AlreadyCancelledRefused(t *testing.T) {
	s := &Service{DB: db.OpenTest(t)}
	before := readCancelRow(t, s.DB, "a2")
	err := s.CancelWithCode(context.Background(), Caller{"u1", "f1"}, "a2", "patient_request", "a second reason")
	if !errors.Is(err, ErrTerminal) {
		t.Fatalf("want ErrTerminal, got %v", err)
	}
	got := readCancelRow(t, s.DB, "a2")
	if got != before || got.Reason != "patient called, travelling" {
		t.Fatalf("a2 changed: %+v -> %+v", before, got)
	}
}

func TestAC5_UnknownAppointmentNotFound(t *testing.T) {
	s := &Service{DB: db.OpenTest(t)}
	err := s.CancelWithCode(context.Background(), Caller{"u1", "f1"}, "nope", "patient_request", "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

// Two staff cancel the same appointment at once: exactly one wins, and the
// other is told the appointment is already cancelled instead of success.
func TestAC5_ConcurrentCancelOnlyOneSucceeds(t *testing.T) {
	d := db.OpenTest(t)
	d.SetMaxOpenConns(1)
	s := &Service{DB: d}
	codes := [2]string{"patient_request", "provider_unavailable"}
	for i := 0; i < 200; i++ {
		if _, err := d.Exec(`UPDATE appointments SET status = 'booked', cancel_reason = '', cancel_reason_code = '', cancel_notes = NULL WHERE id = 'a1'`); err != nil {
			t.Fatal(err)
		}
		var errs [2]error
		var wg sync.WaitGroup
		for g := range codes {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs[g] = s.CancelWithCode(context.Background(), Caller{"u1", "f1"}, "a1", codes[g], "")
			}()
		}
		wg.Wait()
		winner := -1
		for g, err := range errs {
			switch {
			case err == nil && winner == -1:
				winner = g
			case errors.Is(err, ErrTerminal):
			default:
				t.Fatalf("iteration %d: errs %v", i, errs)
			}
		}
		if winner == -1 {
			t.Fatalf("iteration %d: nobody won: %v", i, errs)
		}
		if got := readCancelRow(t, d, "a1"); got.Code != codes[winner] {
			t.Fatalf("iteration %d: winner stored %q, row has %+v", i, codes[winner], got)
		}
	}
}
