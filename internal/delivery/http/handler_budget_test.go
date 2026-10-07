package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"github.com/Damirka228/travel_aggregator/internal/usecase"
	"github.com/rs/zerolog"
)

func TestDestinationsHandler_BudgetValidation(t *testing.T) {
	for _, tc := range []struct {
		name, budget string
		status       int
	}{
		{"missing", "", http.StatusBadRequest},
		{"malformed", "abc", http.StatusBadRequest},
		{"overflow", "1e9999", http.StatusBadRequest},
		{"NaN", "NaN", http.StatusBadRequest},
		{"positive infinity", "+Inf", http.StatusBadRequest},
		{"negative infinity", "-Inf", http.StatusBadRequest},
		{"infinity alias", "Infinity", http.StatusBadRequest},
		{"negative", "-5000", http.StatusBadRequest},
		{"zero", "0", http.StatusBadRequest},
		{"below minimum", "999.99", http.StatusBadRequest},
		{"exact minimum", "1000", http.StatusBadRequest},
		{"above minimum", "1000.01", http.StatusOK},
		{"regular", "100000", http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			source := handlerDateSource(func(context.Context, domain.FlightSearch) ([]domain.Destination, error) {
				calls.Add(1)
				return nil, nil
			})
			service := usecase.NewTravelService(logger.Logger{Logger: zerolog.Nop()}, &mockHotelRepo{}, source)
			params := url.Values{"budget": {tc.budget}, "days": {"7"}, "city": {"MOW"}, "departure_date": {"2026-11-10"}}
			request := httptest.NewRequest(http.MethodGet, "/destinations?"+params.Encode(), nil)
			response := httptest.NewRecorder()
			NewHandler(service).Destinations(response, request)
			if response.Code != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, response.Code, response.Body.String())
			}
			wantCalls := int32(1)
			if tc.status == http.StatusBadRequest {
				wantCalls = 0
				if !strings.Contains(response.Body.String(), "invalid budget") {
					t.Fatalf("expected a budget error: %s", response.Body.String())
				}
			}
			if calls.Load() != wantCalls {
				t.Fatalf("expected %d source calls, got %d", wantCalls, calls.Load())
			}
		})
	}
}
