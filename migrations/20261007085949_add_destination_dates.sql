-- +goose Up
ALTER TABLE destinations
    ADD COLUMN origin TEXT,
    ADD COLUMN departure_date DATE,
    ADD COLUMN return_date DATE,
    ADD COLUMN departure_at TIMESTAMPTZ,
    ADD COLUMN return_at TIMESTAMPTZ,
    ADD CONSTRAINT destinations_date_order
        CHECK (return_date > departure_date);

-- +goose Down
ALTER TABLE destinations
    DROP CONSTRAINT destinations_date_order,
    DROP COLUMN return_at,
    DROP COLUMN departure_at,
    DROP COLUMN return_date,
    DROP COLUMN departure_date,
    DROP COLUMN origin;