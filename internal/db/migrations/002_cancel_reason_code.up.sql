ALTER TABLE appointments ADD COLUMN cancel_reason_code TEXT NOT NULL DEFAULT '';
ALTER TABLE appointments ADD COLUMN cancel_notes TEXT;
