package scheduling

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// CancelReasons maps each allowed cancel reason code to its label.
var CancelReasons = map[string]string{
	"patient_request":      "Patient requested cancellation",
	"provider_unavailable": "Provider unavailable",
	"clinic_closure":       "Clinic or service closed",
	"booked_in_error":      "Booked in error",
	"other":                "Other",
}

// CancelWithCode cancels a booked appointment with a coded reason and an optional note.
func (s *Service) CancelWithCode(ctx context.Context, c Caller, id, code, note string) error {
	label, ok := CancelReasons[code]
	if !ok {
		return ErrInvalid
	}
	var status string
	err := s.DB.QueryRowContext(ctx, `SELECT status FROM appointments WHERE id = ?`, id).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status == "cancelled" {
		return ErrTerminal
	}
	var notes any
	if n := strings.TrimSpace(note); n != "" {
		notes = n
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE appointments SET status = 'cancelled', cancel_reason = ?, cancel_reason_code = ?, cancel_notes = ? WHERE id = ? AND status <> 'cancelled'`, label, code, notes, id)
	return err
}
