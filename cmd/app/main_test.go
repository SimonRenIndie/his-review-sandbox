package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/SimonRenIndie/his-review-sandbox/internal/db"
	"github.com/SimonRenIndie/his-review-sandbox/internal/scheduling"
)

type row struct {
	Status, Reason, Code string
	Notes                sql.NullString
}

func readRow(t *testing.T, d *sql.DB, id string) row {
	t.Helper()
	var r row
	err := d.QueryRow(`SELECT status, cancel_reason, cancel_reason_code, cancel_notes FROM appointments WHERE id = ?`, id).
		Scan(&r.Status, &r.Reason, &r.Code, &r.Notes)
	if err != nil {
		t.Fatalf("read %s: %v", id, err)
	}
	return r
}

func post(h http.Handler, path, facility string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-User", "u1")
	req.Header.Set("X-Facility", facility)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAC3_CodedRouteCancels(t *testing.T) {
	d := db.OpenTest(t)
	h := newMux(&scheduling.Service{DB: d})
	rec := post(h, "/appointments/cancel-coded", "f1", url.Values{"id": {"a1"}, "code": {"provider_unavailable"}, "note": {"  surgeon off sick  "}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	want := row{Status: "cancelled", Reason: "Provider unavailable", Code: "provider_unavailable", Notes: sql.NullString{String: "surgeon off sick", Valid: true}}
	if got := readRow(t, d, "a1"); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestAC5_P1_CodedRouteUsesHeaderFacility(t *testing.T) {
	d := db.OpenTest(t)
	h := newMux(&scheduling.Service{DB: d})
	before := readRow(t, d, "a3")
	rec := post(h, "/appointments/cancel-coded", "f1", url.Values{"id": {"a3"}, "code": {"patient_request"}, "facility": {"f2"}})
	if rec.Code == http.StatusOK {
		t.Fatalf("f1 cancelled f2's a3")
	}
	if got := readRow(t, d, "a3"); got != before {
		t.Fatalf("a3 changed: %+v -> %+v", before, got)
	}
}

func TestAC1_CodedRouteRefusesUnknownCode(t *testing.T) {
	d := db.OpenTest(t)
	h := newMux(&scheduling.Service{DB: d})
	before := readRow(t, d, "a1")
	rec := post(h, "/appointments/cancel-coded", "f1", url.Values{"id": {"a1"}, "code": {"bored"}})
	if rec.Code == http.StatusOK {
		t.Fatalf("unknown code accepted")
	}
	if got := readRow(t, d, "a1"); got != before {
		t.Fatalf("a1 changed: %+v -> %+v", before, got)
	}
}

func TestFreeTextRouteUnchanged(t *testing.T) {
	d := db.OpenTest(t)
	h := newMux(&scheduling.Service{DB: d})
	rec := post(h, "/appointments/cancel", "f1", url.Values{"id": {"a1"}, "reason": {"patient requested"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	want := row{Status: "cancelled", Reason: "patient requested"}
	if got := readRow(t, d, "a1"); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
