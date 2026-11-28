package scheduling

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/SimonRenIndie/his-review-sandbox/internal/db"
)

func TestAC1_AC2_CountsByCodeForOwnFacility(t *testing.T) {
	s := &Service{DB: db.OpenTest(t)}
	ctx := context.Background()
	if err := s.CancelWithCode(ctx, Caller{"u1", "f1"}, "a1", "clinic_closure", ""); err != nil {
		t.Fatal(err)
	}
	got, err := s.CancellationCounts(ctx, Caller{"u1", "f1"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"clinic_closure": 1, "uncoded": 1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestAC1_OtherFacilityNotCounted(t *testing.T) {
	s := &Service{DB: db.OpenTest(t)}
	got, err := s.CancellationCounts(context.Background(), Caller{"u2", "f2"})
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]int{"uncoded": 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestAC3_NoFacilityRefused(t *testing.T) {
	s := &Service{DB: db.OpenTest(t)}
	if _, err := s.CancellationCounts(context.Background(), Caller{}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
}
