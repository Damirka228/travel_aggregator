package usecase

import (
	"context"
	"testing"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
)

type mockDestinationRepository struct{}

func (m *mockDestinationRepository) GetAll(ctx context.Context, origin string) ([]domain.Destination, error) {
	var result []domain.Destination
	for i := 1; i <= 100; i++ {
		result = append(result, domain.Destination{
			ID: i, City: "KZN", Country: "International", Price: float64(2000 + (i * 100)), Days: 3,
		})
	}
	return result, nil
}

func BenchmarkFindDestinations(b *testing.B) {
	log := logger.New()
	mockFlight := &mockDestinationRepository{}
	mockHotel := &mockHotelRepository{}

	service := NewTravelService(log, mockHotel, mockFlight)
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = service.FindDestinations(ctx, 300000, 7, "MOW")
	}
}
