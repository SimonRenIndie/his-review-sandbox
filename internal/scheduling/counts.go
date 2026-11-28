package scheduling

import (
	"context"
	"strings"
)

// CancellationCounts returns how many of the caller's facility's appointments
// were cancelled with each reason code. Cancellations made before reason codes
// existed are counted under "uncoded".
func (s *Service) CancellationCounts(ctx context.Context, c Caller) (map[string]int, error) {
	if strings.TrimSpace(c.FacilityID) == "" {
		return nil, ErrForbidden
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT CASE WHEN cancel_reason_code = '' THEN 'uncoded' ELSE cancel_reason_code END, count(*)
		FROM appointments
		WHERE facility_id = ? AND status = 'cancelled'
		GROUP BY 1`, c.FacilityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var code string
		var n int
		if err := rows.Scan(&code, &n); err != nil {
			return nil, err
		}
		counts[code] = n
	}
	return counts, rows.Err()
}
