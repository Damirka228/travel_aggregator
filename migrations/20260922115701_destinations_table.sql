-- +goose Up
SELECT 'up SQL query';

CREATE TABLE destinations(
    id SERIAL PRIMARY KEY,
    city TEXT NOT NULL,
    country TEXT NOT NULL,
    price NUMERIC(10,2) NOT NULL,
    days INTEGER NOT NULL
);

-- +goose Down
SELECT 'down SQL query';
