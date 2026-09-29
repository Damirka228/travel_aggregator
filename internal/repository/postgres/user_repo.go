package postgres

import (
	"context"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/repository/dbqueries"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepository struct {
	queries *dbqueries.Queries
}

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{
		queries: dbqueries.New(pool),
	}
}

func (r *PostgresUserRepository) CreateUser(ctx context.Context, email string, hashPass string) (domain.User, error) {
	dbUser, err := r.queries.CreateUser(ctx, dbqueries.CreateUserParams{
		Email:        email,
		PasswordHash: hashPass,
	})
	if err != nil {
		return domain.User{}, err
	}

	return domain.User{
		ID:           int(dbUser.ID),
		Email:        dbUser.Email,
		PasswordHash: hashPass,
	}, nil
}

func (r *PostgresUserRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	dbUser, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{
		ID:           int(dbUser.ID),
		Email:        dbUser.Email,
		PasswordHash: dbUser.PasswordHash,
	}, nil
}
