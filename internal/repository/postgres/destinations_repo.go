package postgres

import (
	"context"
	"fmt"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/repository/dbqueries"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDestinationRepository struct {
	queries *dbqueries.Queries
}

func NewPostgresDestinationRepository(pool *pgxpool.Pool) *PostgresDestinationRepository {
	return &PostgresDestinationRepository{queries: dbqueries.New(pool)}
}

func (r *PostgresDestinationRepository) GetAll(ctx context.Context, search domain.FlightSearch) ([]domain.Destination, error) {
	params := dbqueries.GetAllDestinationsParams{
		Origin: search.Origin,
		DepartureDate: pgtype.Date{
			Time:  search.DepartureDate,
			Valid: true,
		},
		ReturnDate: pgtype.Date{
			Time:  search.ReturnDate,
			Valid: true,
		},
	}

	rows, err := r.queries.GetAllDestinations(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("get destinations from postgres: %w", err)
	}

	result := make([]domain.Destination, 0, len(rows))

	for _, row := range rows {
		price, err := row.Price.Float64Value()
		if err != nil {
			return nil, fmt.Errorf("convert price for destination %d: %w", row.ID, err)
		}

		result = append(result, domain.Destination{
			ID:          int(row.ID),
			Origin:      row.Origin.String,
			City:        row.City,
			Country:     row.Country,
			Price:       price.Float64,
			Days:        int(row.Days),
			DepartureAt: row.DepartureAt.Time,
			ReturnAt:    row.ReturnAt.Time,
		})
	}

	return result, nil
}
