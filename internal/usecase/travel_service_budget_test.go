package usecase

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"github.com/rs/zerolog"
)

func TestFindDestinations_BudgetValidation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		budget  float64
		wantErr error
	}{
		{"NaN", math.NaN(), ErrInvalidBudget},
		{"positive infinity", math.Inf(1), ErrInvalidBudget},
		{"negative infinity", math.Inf(-1), ErrInvalidBudget},
		{"negative budget", -5000, ErrInvalidBudget},
		{"zero", 0, ErrInvalidBudget},
		{"below minimum", 999.99, ErrInvalidBudget},
		{"exact minimum", 1000, ErrInvalidBudget},
		{"immediately above minimum", math.Nextafter(1000, math.Inf(1)), nil},
		{"fractional budget", 1000.01, nil},
		{"regular budget", 100000, nil},
		{"largest finite budget", math.MaxFloat64, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &availabilityFlightRepository{}
			service := NewTravelService(logger.Logger{Logger: zerolog.Nop()}, &mockHotelRepository{}, source)
			result, err := service.FindDestinations(context.Background(), tc.budget, 7, "MOW", travelTestDate())
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
			wantCalls := int32(1)
			if tc.wantErr != nil {
				wantCalls = 0
			}
			if source.calls.Load() != wantCalls {
				t.Fatalf("expected %d source calls, got %d", wantCalls, source.calls.Load())
			}
			if len(result.BestTour) != 0 || len(result.Cheapest) != 0 || len(result.Longest) != 0 {
				t.Fatalf("expected empty result from empty source: %+v", result)
			}
		})
	}
}
