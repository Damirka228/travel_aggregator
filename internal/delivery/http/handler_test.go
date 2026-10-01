package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"github.com/Damirka228/travel_aggregator/internal/usecase"
)

type mockFlightRepository struct{}

func (m *mockFlightRepository) GetAll(ctx context.Context, origin string) ([]domain.Destination, error) {
	return []domain.Destination{
		{ID: 1, City: "KZN", Country: "International", Price: 5000, Days: 3},
	}, nil
}

func TestDestinationsHandler_Success(t *testing.T) {
	log := logger.New()
	mockRepo := &mockFlightRepository{}
	travelService := usecase.NewTravelService(log, mockRepo)
	handler := NewHandler(travelService)

	req, err := http.NewRequest("GET", "/destinations?budget=100000&days=5&city=MOW", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.Destinations(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Ожидался статус 200, получено: %v", status)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Ожидался JSON-ответ, получено: %v", contentType)
	}
}

func TestDestinationsHandler_TooSmallBudget(t *testing.T) {
	log := logger.New()
	mockRepo := &mockFlightRepository{}
	travelService := usecase.NewTravelService(log, mockRepo)
	handler := NewHandler(travelService)

	req, err := http.NewRequest("GET", "/destinations?budget=500&days=5&city=MOW", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	handler.Destinations(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Ожидался статус 400 при бюджете <= 1000, получено: %v", status)
	}
}

func TestDestinationsHandler_MissingCity(t *testing.T) {
	log := logger.New()
	mockRepo := &mockFlightRepository{}
	travelService := usecase.NewTravelService(log, mockRepo)
	handler := NewHandler(travelService)

	req, err := http.NewRequest("GET", "/destinations?budget=50000&days=5", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	handler.Destinations(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Ожидался статус 400 при отсутствии города, получено: %v", status)
	}
}
