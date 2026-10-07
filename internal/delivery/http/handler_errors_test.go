package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"github.com/Damirka228/travel_aggregator/internal/usecase"
	"github.com/rs/zerolog"
)

type availabilityHTTPSource struct {
	err error
}

func (r *availabilityHTTPSource) GetAll(context.Context, domain.FlightSearch) ([]domain.Destination, error) {
	return nil, r.err
}

func TestDestinationsHandler_ValidationAndAvailability(t *testing.T) {
	failed := errors.New("upstream unavailable")
	for _, tc := range []struct {
		name     string
		days     string
		sources  []domain.DestinationRepository
		status   int
		wantTour bool
	}{
		{"negative days", "-3", []domain.DestinationRepository{&mockFlightRepo{}}, http.StatusBadRequest, false},
		{"zero days", "0", []domain.DestinationRepository{&mockFlightRepo{}}, http.StatusBadRequest, false},
		{"invalid days format", "abc", []domain.DestinationRepository{&mockFlightRepo{}}, http.StatusBadRequest, false},
		{"all sources failed", "3", []domain.DestinationRepository{&availabilityHTTPSource{err: failed}, &availabilityHTTPSource{err: failed}}, http.StatusServiceUnavailable, false},
		{"partial failure", "3", []domain.DestinationRepository{&availabilityHTTPSource{err: failed}, &mockFlightRepo{}}, http.StatusOK, true},
		{"successful empty source", "3", []domain.DestinationRepository{&availabilityHTTPSource{err: failed}, &availabilityHTTPSource{}}, http.StatusOK, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := usecase.NewTravelService(logger.Logger{Logger: zerolog.Nop()}, &mockHotelRepo{}, tc.sources...)
			handler := NewHandler(service)
			request := httptest.NewRequest(http.MethodGet, "/destinations?budget=100000&days="+tc.days+"&city=MOW&departure_date=2026-11-10", nil)
			response := httptest.NewRecorder()
			handler.Destinations(response, request)
			if response.Code != tc.status {
				t.Fatalf("expected status %d, got %d: %s", tc.status, response.Code, response.Body.String())
			}
			if tc.status == http.StatusOK {
				if got := response.Header().Get("Content-Type"); got != "application/json" {
					t.Fatalf("expected JSON content type, got %q", got)
				}
				var result usecase.TravelResponse
				if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
					t.Fatalf("invalid JSON response: %v", err)
				}
				if got := len(result.BestTour) > 0; got != tc.wantTour {
					t.Fatalf("expected tour=%v, got %+v", tc.wantTour, result)
				}
			}
		})
	}
}
