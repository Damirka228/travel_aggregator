-- name: GetAllDestinations :many
SELECT
    id,
    origin,
    city,
    country,
    price,
    (return_date - departure_date)::integer AS days,
    departure_at,
    return_at
FROM destinations
WHERE origin = sqlc.arg(origin)::text
  AND departure_date = sqlc.arg(departure_date)::date
  AND return_date = sqlc.arg(return_date)::date
  AND departure_at IS NOT NULL
  AND return_at IS NOT NULL
ORDER BY price ASC, id ASC;