//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostgresDestinationsRepository(t *testing.T) {
	ctx := context.Background()
	container, err := postgres.RunContainer(ctx, testcontainers.WithImage("postgres:16"))
	require.NoError(t, err, "не удалось запустить контейнер")
	t.Cleanup(func() { container.Terminate(ctx) })
	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "не удалось получить строку подключения")
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err, "не удалось получить пул подключений")

	var pingErr error
	for i := 0; i < 10; i++ {
		pingErr = pool.Ping(ctx)
		if pingErr == nil {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	require.NoError(t, pingErr, "база не отвечает после ретраев")

	query := `
	CREATE TABLE destinations (
		id SERIAL PRIMARY KEY, 
		city TEXT NOT NULL, 
		country TEXT NOT NULL, 
		price NUMERIC(10,2) NOT NULL, 
		days INTEGER NOT NULL);`
	_, err = pool.Exec(ctx, query)
	require.NoError(t, err, "не удалось создать таблицу destinations")

	queryINTO := `
    INSERT INTO destinations (city, country, price, days) VALUES
    ('Дубай', 'ОАЭ', 350000, 8),
    ('Барселона', 'Испания', 280000, 9);`
	tag, err := pool.Exec(ctx, queryINTO)
	require.NoError(t, err, "ошибка добавления направлений в базу данных")
	require.Equal(t, int64(2), tag.RowsAffected())

	repo := NewPostgresDestinationRepository(pool)

	result, err := repo.GetAll(ctx)
	assert.NoError(t, err, "ошибка получения результата")
	assert.Len(t, result, 2, "неккореткное число направлений")
	var cities []string
	for _, v := range result {
		cities = append(cities, v.City)
	}
	assert.ElementsMatch(t, []string{"Дубай", "Барселона"}, cities)
}
