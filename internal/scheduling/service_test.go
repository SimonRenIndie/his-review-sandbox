package scheduling

import (
	"context"
	"errors"
	"testing"

	"github.com/SimonRenIndie/his-review-sandbox/internal/db"
)

func TestCancel_StoresReason(t *testing.T) {
	s := &Service{DB: db.OpenTest(t)}
	if err := s.Cancel(context.Background(), Caller{"u1", "f1"}, "a1", "patient requested"); err != nil {
		t.Fatal(err)
	}
	var st, r string
	s.DB.QueryRow(`SELECT status, cancel_reason FROM appointments WHERE id='a1'`).Scan(&st, &r)
	if st != "cancelled" || r != "patient requested" {
		t.Fatalf("got %q %q", st, r)
	}
}

func TestCancel_OtherFacilityRefused(t *testing.T) {
	s := &Service{DB: db.OpenTest(t)}
	err := s.Cancel(context.Background(), Caller{"u1", "f1"}, "a3", "patient requested")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
	var st string
	s.DB.QueryRow(`SELECT status FROM appointments WHERE id='a3'`).Scan(&st)
	if st != "booked" {
		t.Fatalf("a3 changed to %q", st)
	}
}

func TestCancel_AlreadyCancelledRefused(t *testing.T) {
	s := &Service{DB: db.OpenTest(t)}
	err := s.Cancel(context.Background(), Caller{"u1", "f1"}, "a2", "again")
	if !errors.Is(err, ErrTerminal) {
		t.Fatalf("want ErrTerminal, got %v", err)
	}
	var r string
	s.DB.QueryRow(`SELECT cancel_reason FROM appointments WHERE id='a2'`).Scan(&r)
	if r != "patient called, travelling" {
		t.Fatalf("first reason overwritten: %q", r)
	}
}

func TestCancel_EmptyReasonRefused(t *testing.T) {
	s := &Service{DB: db.OpenTest(t)}
	if err := s.Cancel(context.Background(), Caller{"u1", "f1"}, "a1", "  "); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}
