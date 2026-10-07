package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"github.com/Damirka228/travel_aggregator/internal/usecase"
	"github.com/rs/zerolog"
)

type mockFlightRepo struct{}

func (m *mockFlightRepo) GetAll(ctx context.Context, search domain.FlightSearch) ([]domain.Destination, error) {
	return []domain.Destination{
		{ID: 1, Origin: search.Origin, City: "KZN", Country: "International", Price: 5000, Days: int(search.ReturnDate.Sub(search.DepartureDate) / (24 * time.Hour)), DepartureAt: search.DepartureDate, ReturnAt: search.ReturnDate},
	}, nil
}

type mockHotelRepo struct{}

func (m *mockHotelRepo) GetByCity(ctx context.Context, city string) ([]domain.Hotel, error) {
	return []domain.Hotel{
		{ID: 201, Name: "Test Hotel", City: city, Price: 1000, Stars: 3},
	}, nil
}

func TestDestinationsHandler_Success(t *testing.T) {
	log := logger.Logger{Logger: zerolog.Nop()}
	mockFlight := &mockFlightRepo{}
	mockHotel := &mockHotelRepo{}
	travelService := usecase.NewTravelService(log, mockHotel, mockFlight)
	handler := NewHandler(travelService)

	req, err := http.NewRequest("GET", "/destinations?budget=100000&departure_date=2026-11-10&days=5&city=MOW", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.Destinations(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status: got %v, want %v", status, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected JSON content type, got: %v", contentType)
	}
}

func TestDestinationsHandler_TooSmallBudget(t *testing.T) {
	log := logger.Logger{Logger: zerolog.Nop()}
	mockFlight := &mockFlightRepo{}
	mockHotel := &mockHotelRepo{}
	travelService := usecase.NewTravelService(log, mockHotel, mockFlight)
	handler := NewHandler(travelService)

	req, err := http.NewRequest("GET", "/destinations?budget=500&departure_date=2026-11-10&days=5&city=MOW", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	handler.Destinations(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status 400 for budget <= 1000, got: %v", status)
	}
}

func TestDestinationsHandler_MissingCity(t *testing.T) {
	log := logger.Logger{Logger: zerolog.Nop()}
	mockFlight := &mockFlightRepo{}
	mockHotel := &mockHotelRepo{}
	travelService := usecase.NewTravelService(log, mockHotel, mockFlight)
	handler := NewHandler(travelService)

	req, err := http.NewRequest("GET", "/destinations?budget=50000&departure_date=2026-11-10&days=5", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	handler.Destinations(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing city, got: %v", status)
	}
}
