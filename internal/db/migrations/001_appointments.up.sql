CREATE TABLE appointments (
    id            TEXT PRIMARY KEY,
    facility_id   TEXT NOT NULL,
    patient_name  TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'booked',
    cancel_reason TEXT NOT NULL DEFAULT ''
);
