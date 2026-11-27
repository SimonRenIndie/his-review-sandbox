package scheduling

import (
	"context"
	"errors"
	"testing"

	"github.com/SimonRenIndie/his-review-sandbox/internal/db"
)

func TestAC1_UnknownCodeRefused(t *testing.T) {
	s := &Service{DB: db.OpenTest(t)}
	err := s.CancelWithCode(context.Background(), Caller{"u1", "f1"}, "a1", "bored", "")
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
	var st string
	s.DB.QueryRow(`SELECT status FROM appointments WHERE id='a1'`).Scan(&st)
	if st != "booked" {
		t.Fatalf("a1 changed to %q", st)
	}
}

func TestAC2_OtherNeedsNote(t *testing.T) {
	s := &Service{DB: db.OpenTest(t)}
	if err := s.Cancel(context.Background(), Caller{"u1", "f1"}, "a1", ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func TestAC3_StoresCodeLabelAndNote(t *testing.T) {
	s := &Service{DB: db.OpenTest(t)}
	if err := s.CancelWithCode(context.Background(), Caller{"u1", "f1"}, "a1", "clinic_closure", " power cut "); err != nil {
		t.Fatal(err)
	}
	var code, reason, notes string
	s.DB.QueryRow(`SELECT cancel_reason_code, cancel_reason, cancel_notes FROM appointments WHERE id='a1'`).Scan(&code, &reason, &notes)
	if code != "clinic_closure" || reason != "Clinic or service closed" || notes != "power cut" {
		t.Fatalf("got %q %q %q", code, reason, notes)
	}
}

func TestAC4_LegacyRowsHaveNoCode(t *testing.T) {
	s := &Service{DB: db.OpenTest(t)}
	var code string
	s.DB.QueryRow(`SELECT cancel_reason_code FROM appointments WHERE id='a2'`).Scan(&code)
	if code != "" {
		t.Fatalf("legacy row got code %q", code)
	}
}
