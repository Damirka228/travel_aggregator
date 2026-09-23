package postgres

import (
	"context"
	"fmt"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/repository/dbqueries"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDestinationRepository struct {
	queries *dbqueries.Queries
}

func NewPostgresDestinationRepository(pool *pgxpool.Pool) *PostgresDestinationRepository {
	return &PostgresDestinationRepository{queries: dbqueries.New(pool)}
}

func (r *PostgresDestinationRepository) GetAll(ctx context.Context) ([]domain.Destination, error) {
	rows, err := r.queries.GetAllDestinations(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Destination, 0, len(rows))

	for _, row := range rows {
		price, err := row.Price.Float64Value()
		if err != nil {
			return nil, fmt.Errorf("конвертация цена для направления %d: %w", row.ID, err)
		}

		result = append(result, domain.Destination{
			ID:      int(row.ID),
			City:    row.City,
			Country: row.Country,
			Price:   price.Float64,
			Days:    int(row.Days),
		})
	}
	return result, nil
}
