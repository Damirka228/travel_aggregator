//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresDestinations_DateSearchIntegration(t *testing.T) {
	databaseURL := os.Getenv("TRAVEL_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TRAVEL_TEST_DATABASE_URL to run the PostgreSQL integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	// The temporary table belongs to one session; all queries use that session.
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	// Copy the migrated schema and check constraints, then shadow the public table.
	_, err = pool.Exec(ctx, `CREATE TEMP TABLE destinations (LIKE public.destinations INCLUDING CONSTRAINTS)`)
	if err != nil {
		t.Fatalf("create isolated fixture table (apply migrations first): %v", err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO pg_temp.destinations
		(id, city, country, price, days, origin, departure_date, return_date, departure_at, return_at)
		VALUES
		(1, 'KZN', 'RU', 9000, 99, 'MOW', '2026-11-10', '2026-11-17', '2026-11-10T23:00:00+03:00', '2026-11-17T08:00:00+03:00'),
		(2, 'KZN', 'RU', 9000, 99, 'LED', '2026-11-10', '2026-11-17', '2026-11-10T23:00:00+03:00', '2026-11-17T08:00:00+03:00'),
		(3, 'KZN', 'RU', 9000, 99, 'MOW', '2026-11-11', '2026-11-17', '2026-11-11T23:00:00+03:00', '2026-11-17T08:00:00+03:00'),
		(4, 'KZN', 'RU', 9000, 99, 'MOW', '2026-11-10', '2026-11-18', '2026-11-10T23:00:00+03:00', '2026-11-18T08:00:00+03:00'),
		(5, 'KZN', 'RU', 1, 7, NULL, NULL, NULL, NULL, NULL),
		(6, 'KZN', 'RU', 1, 7, 'MOW', '2026-11-10', '2026-11-17', '2026-11-10T23:00:00+03:00', NULL),
		(7, 'KZN', 'RU', 5000, 99, 'MOW', '2026-11-10', '2026-11-17', '2026-11-10T23:00:00+03:00', '2026-11-17T08:00:00+03:00'),
		(8, 'KZN', 'RU', 9000, 99, 'MOW', '2026-11-10', '2026-11-17', '2026-11-10T23:00:00+03:00', '2026-11-17T08:00:00+03:00')`)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresDestinationRepository(pool)
	for _, tc := range []struct {
		name, origin, departure, returning string
		ids                                []int
		days                               int
	}{
		{"matching dates sorted by price then ID", "MOW", "2026-11-10", "2026-11-17", []int{7, 1, 8}, 7},
		{"different origin", "LED", "2026-11-10", "2026-11-17", []int{2}, 7},
		{"different departure", "MOW", "2026-11-11", "2026-11-17", []int{3}, 6},
		{"different return", "MOW", "2026-11-10", "2026-11-18", []int{4}, 8},
		{"no matching offers", "MOW", "2026-12-10", "2026-12-17", nil, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			departure, _ := time.Parse("2006-01-02", tc.departure)
			returning, _ := time.Parse("2006-01-02", tc.returning)
			got, err := repo.GetAll(ctx, domain.FlightSearch{Origin: tc.origin, DepartureDate: departure, ReturnDate: returning})
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tc.ids) {
				t.Fatalf("wrong selected offers: %+v", got)
			}
			for i, offer := range got {
				if offer.ID != tc.ids[i] || offer.Origin != tc.origin || offer.City != "KZN" || offer.Country != "RU" || offer.Days != tc.days {
					t.Fatalf("wrong offer mapping: %+v", offer)
				}
				wantPrice := float64(9000)
				if offer.ID == 7 {
					wantPrice = 5000
				}
				wantDeparture, _ := time.Parse(time.RFC3339, tc.departure+"T23:00:00+03:00")
				wantReturn, _ := time.Parse(time.RFC3339, tc.returning+"T08:00:00+03:00")
				if offer.Price != wantPrice || !offer.DepartureAt.Equal(wantDeparture) || !offer.ReturnAt.Equal(wantReturn) {
					t.Fatalf("price or flight instants were lost: %+v", offer)
				}
			}
		})
	}
	for _, returning := range []string{"2026-11-09", "2026-11-10"} {
		t.Run("constraint rejects return "+returning, func(t *testing.T) {
			_, err := pool.Exec(ctx, `INSERT INTO pg_temp.destinations (id, city, country, price, days, origin, departure_date, return_date) VALUES (99, 'KZN', 'RU', 5000, 7, 'MOW', '2026-11-10', $1::date)`, returning)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
				t.Fatalf("expected check constraint violation, got %v", err)
			}
		})
	}
}
