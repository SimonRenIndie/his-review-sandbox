package scheduling

import (
	"context"
	"strings"
	"unicode/utf8"
)

// CancelReasons maps each allowed cancel reason code to its label.
var CancelReasons = map[string]string{
	"patient_request":      "Patient requested cancellation",
	"provider_unavailable": "Provider unavailable",
	"clinic_closure":       "Clinic or service closed",
	"booked_in_error":      "Booked in error",
	"other":                "Other",
}

// minOtherNoteRunes is the shortest trimmed note accepted with code "other".
const minOtherNoteRunes = 10

// CancelWithCode cancels a booked appointment with a coded reason and an optional note.
func (s *Service) CancelWithCode(ctx context.Context, c Caller, id, code, note string) error {
	label, ok := CancelReasons[code]
	if !ok {
		return ErrInvalid
	}
	note = strings.TrimSpace(note)
	if code == "other" && utf8.RuneCountInString(note) < minOtherNoteRunes {
		return ErrInvalid
	}
	status, err := s.authorize(ctx, c, id)
	if err != nil {
		return err
	}
	if status == "cancelled" {
		return ErrTerminal
	}
	var notes any
	if note != "" {
		notes = note
	}
	res, err := s.DB.ExecContext(ctx, `UPDATE appointments SET status = 'cancelled', cancel_reason = ?, cancel_reason_code = ?, cancel_notes = ? WHERE id = ? AND status <> 'cancelled'`, label, code, notes, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		// Another cancel landed between authorize and this update.
		return ErrTerminal
	}
	return nil
}
