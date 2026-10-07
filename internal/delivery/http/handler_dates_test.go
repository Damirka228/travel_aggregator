package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"github.com/Damirka228/travel_aggregator/internal/usecase"
	"github.com/rs/zerolog"
)

type handlerDateSource func(context.Context, domain.FlightSearch) ([]domain.Destination, error)

func (f handlerDateSource) GetAll(ctx context.Context, search domain.FlightSearch) ([]domain.Destination, error) {
	return f(ctx, search)
}

func TestDestinationsHandler_InvalidDepartureDate(t *testing.T) {
	for _, value := range []string{"", "2026/11/10", "2026-02-30", "2026-11-10T09:30:00Z"} {
		t.Run(value, func(t *testing.T) {
			var calls atomic.Int32
			source := handlerDateSource(func(context.Context, domain.FlightSearch) ([]domain.Destination, error) {
				calls.Add(1)
				return nil, nil
			})
			service := usecase.NewTravelService(logger.Logger{Logger: zerolog.Nop()}, &mockHotelRepo{}, source)
			request := httptest.NewRequest(http.MethodGet, "/destinations?budget=100000&days=7&city=MOW&departure_date="+url.QueryEscape(value), nil)
			response := httptest.NewRecorder()
			NewHandler(service).Destinations(response, request)
			if response.Code != http.StatusBadRequest || calls.Load() != 0 {
				t.Fatalf("invalid date reached a source: status=%d calls=%d", response.Code, calls.Load())
			}
		})
	}
}

func TestDestinationsHandler_ForwardsDateAndReturnsTour(t *testing.T) {
	seen := make(chan domain.FlightSearch, 1)
	source := handlerDateSource(func(_ context.Context, search domain.FlightSearch) ([]domain.Destination, error) {
		seen <- search
		return []domain.Destination{{Origin: search.Origin, City: "KZN", Price: 5000, Days: 7, DepartureAt: search.DepartureDate, ReturnAt: search.ReturnDate}}, nil
	})
	service := usecase.NewTravelService(logger.Logger{Logger: zerolog.Nop()}, &mockHotelRepo{}, source)
	response := httptest.NewRecorder()
	NewHandler(service).Destinations(response, httptest.NewRequest(http.MethodGet, "/destinations?budget=100000&days=7&city=MOW&departure_date=2026-11-10", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", response.Code, response.Body.String())
	}
	select {
	case search := <-seen:
		if search.Origin != "MOW" || search.DepartureDate != time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC) || search.ReturnDate != time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC) {
			t.Fatalf("unexpected search: %+v", search)
		}
	default:
		t.Fatal("no search reached the source")
	}
	var result usecase.TravelResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.BestTour) != 1 || result.BestTour[0].TotalPrice != 12000 || result.BestTour[0].Flight.DepartureAt.IsZero() || result.BestTour[0].Flight.ReturnAt.IsZero() {
		t.Fatalf("expected dated tour costing 5000 + 1000*7: %+v", result)
	}
}
