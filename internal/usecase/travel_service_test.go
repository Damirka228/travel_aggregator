package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/stretchr/testify/assert"
)

type mockDestinationRepository struct {
	GetAllFunc func(ctx context.Context) ([]domain.Destination, error)
}

func (m *mockDestinationRepository) GetAll(ctx context.Context) ([]domain.Destination, error) {
	return m.GetAllFunc(ctx)
}

func TestFindDestinations_MultipleRepos(t *testing.T) {
	repo1 := &mockDestinationRepository{
		GetAllFunc: func(ctx context.Context) ([]domain.Destination, error) {
			return []domain.Destination{{City: "Бали", Price: 120000, Days: 10}}, nil
		},
	}
	repo2 := &mockDestinationRepository{
		GetAllFunc: func(ctx context.Context) ([]domain.Destination, error) {
			return []domain.Destination{{City: "Ереван", Price: 150000, Days: 9}}, nil
		},
	}

	service := NewTravelService(repo1, repo2)
	res, err := service.FindDestinations(context.Background(), 200000, 15)
	assert.NoError(t, err)
	assert.Len(t, res, 2)
}

func TestFindDestinations(t *testing.T) {
	repo := &mockDestinationRepository{
		GetAllFunc: func(ctx context.Context) ([]domain.Destination, error) {
			return []domain.Destination{
				{City: "Стамбул", Country: "Турция", Price: 450000, Days: 7},
				{City: "Бали", Country: "Индонезия", Price: 120000, Days: 10},
				{City: "Тбилиси", Country: "Грузия", Price: 500000, Days: 5},
				{City: "Курск", Country: "Россия", Price: 40000, Days: 7},
			}, nil
		},
	}
	service := NewTravelService(repo)
	tests := []struct {
		name   string
		budget float64
		days   int
		want   []domain.Destination
	}{
		{
			name:   "хватает на все по бюджету",
			budget: 600000,
			days:   15,
			want: []domain.Destination{
				{City: "Стамбул", Country: "Турция", Price: 450000, Days: 7},
				{City: "Бали", Country: "Индонезия", Price: 120000, Days: 10},
				{City: "Тбилиси", Country: "Грузия", Price: 500000, Days: 5},
				{City: "Курск", Country: "Россия", Price: 40000, Days: 7},
			},
		},
		{
			name:   "ожидаем Бали и Курск, но не Стамбул и Тбилиси",
			budget: 150000,
			days:   10,
			want: []domain.Destination{
				{City: "Бали", Country: "Индонезия", Price: 120000, Days: 10},
				{City: "Курск", Country: "Россия", Price: 40000, Days: 7},
			},
		},
		{
			name:   "никуда не едем сидим дома",
			budget: 0,
			days:   0,
			want:   []domain.Destination{},
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			res, err := service.FindDestinations(context.Background(), c.budget, c.days)
			assert.NoError(t, err)
			assert.Equal(t, c.want, res)
		})
	}
}

func TestFindDestinations_SingleRepoFails_ReturnsEmptyNotError(t *testing.T) {
	repo := &mockDestinationRepository{
		GetAllFunc: func(ctx context.Context) ([]domain.Destination, error) {
			return nil, errors.New("db connection failed")
		},
	}
	service := NewTravelService(repo)

	_, err := service.FindDestinations(context.Background(), 100000, 5)

	assert.NoError(t, err)
	assert.Empty(t, err)
}
