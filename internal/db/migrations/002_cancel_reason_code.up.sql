CREATE TABLE appointments_new (
    id                 TEXT PRIMARY KEY,
    facility_id        TEXT NOT NULL,
    patient_name       TEXT NOT NULL,
    status             TEXT NOT NULL DEFAULT 'booked',
    cancel_reason      TEXT NOT NULL DEFAULT '',
    cancel_reason_code TEXT NOT NULL DEFAULT '',
    cancel_notes       TEXT
);
INSERT INTO appointments_new (id, facility_id, patient_name, status)
    SELECT id, facility_id, patient_name, status FROM appointments;
DROP TABLE appointments;
ALTER TABLE appointments_new RENAME TO appointments;
