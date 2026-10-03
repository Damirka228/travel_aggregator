package usecase

import (
	"context"
	"testing"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
)

type testFlightRepository struct {
	flights []domain.Destination
}

func (t *testFlightRepository) GetAll(ctx context.Context, origin string) ([]domain.Destination, error) {
	return t.flights, nil
}

type mockHotelRepository struct{}

func (m *mockHotelRepository) GetByCity(ctx context.Context, city string) ([]domain.Hotel, error) {
	return []domain.Hotel{
		{ID: 201, Name: "Test Hotel 5", City: city, Price: 2000, Stars: 5},
		{ID: 202, Name: "Test Hotel 2", City: city, Price: 500, Stars: 2},
	}, nil
}

func TestFindDestinations_Success(t *testing.T) {
	log := logger.New()
	mockData := []domain.Destination{
		{ID: 1, City: "KZN", Country: "International", Price: 5000, Days: 5},
	}

	flightRepo := &testFlightRepository{flights: mockData}
	hotelRepo := &mockHotelRepository{}
	service := NewTravelService(log, hotelRepo, flightRepo)

	res, err := service.FindDestinations(context.Background(), 100000, 7, "MOW")
	if err != nil {
		t.Fatalf("Expected success, got err: %v", err)
	}

	if len(res.BestTour) == 0 || len(res.Cheapest) == 0 || len(res.Longest) == 0 {
		t.Error("Tours should not be empty")
	}
}

func TestFindDestinations_Empty(t *testing.T) {
	log := logger.New()
	mockData := []domain.Destination{
		{ID: 1, City: "PAR", Country: "International", Price: 90000, Days: 5},
	}
	flightRepo := &testFlightRepository{flights: mockData}
	hotelRepo := &mockHotelRepository{}
	service := NewTravelService(log, hotelRepo, flightRepo)

	res, err := service.FindDestinations(context.Background(), 10000, 7, "MOW")
	if err != nil {
		t.Fatalf("Expected empty response without err, got err: %v", err)
	}

	if len(res.BestTour) != 0 || len(res.Cheapest) != 0 || len(res.Longest) != 0 {
		t.Error("Expected empty result due to budget limits")
	}
}
