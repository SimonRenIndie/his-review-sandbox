// Package scheduling holds appointment rules.
package scheduling

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

var (
	ErrForbidden = errors.New("forbidden")
	ErrNotFound  = errors.New("not found")
	ErrTerminal  = errors.New("appointment already cancelled")
	ErrInvalid   = errors.New("invalid input")
)

// Caller is the signed-in staff member.
type Caller struct {
	UserID     string
	FacilityID string
}

// Service is the only way to read or change appointments.
type Service struct{ DB *sql.DB }

// authorize loads the appointment's facility and status and refuses a caller from another facility.
func (s *Service) authorize(ctx context.Context, c Caller, id string) (status string, err error) {
	var facility string
	err = s.DB.QueryRowContext(ctx, `SELECT facility_id, status FROM appointments WHERE id = ?`, id).Scan(&facility, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if facility != c.FacilityID {
		return "", ErrForbidden
	}
	return status, nil
}

// Cancel cancels a booked appointment with a free-text reason.
func (s *Service) Cancel(ctx context.Context, c Caller, id, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return ErrInvalid
	}
	status, err := s.authorize(ctx, c, id)
	if err != nil {
		return err
	}
	if status == "cancelled" {
		return ErrTerminal
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE appointments SET status = 'cancelled', cancel_reason = ? WHERE id = ? AND status <> 'cancelled'`, reason, id)
	return err
}
