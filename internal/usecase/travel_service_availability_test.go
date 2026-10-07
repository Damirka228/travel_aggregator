package usecase

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"github.com/rs/zerolog"
)

type availabilityFlightRepository struct {
	flights []domain.Destination
	err     error
	calls   atomic.Int32
}

func (r *availabilityFlightRepository) GetAll(_ context.Context, _ domain.FlightSearch) ([]domain.Destination, error) {
	r.calls.Add(1)
	return r.flights, r.err
}

func TestFindDestinations_SourceAvailability(t *testing.T) {
	failed := errors.New("source unavailable")
	flights := []domain.Destination{{City: "KZN", Price: 5000, Days: 3}}
	for _, tc := range []struct {
		name     string
		sources  []domain.DestinationRepository
		wantErr  error
		wantTour bool
	}{
		{"all failed", []domain.DestinationRepository{&availabilityFlightRepository{err: failed}, &availabilityFlightRepository{err: failed}}, ErrFlightSourcesUnavailable, false},
		{"partial failure", []domain.DestinationRepository{&availabilityFlightRepository{err: failed}, &availabilityFlightRepository{flights: flights}}, nil, true},
		{"successful empty source", []domain.DestinationRepository{&availabilityFlightRepository{err: failed}, &availabilityFlightRepository{}}, nil, false},
		{"all successful empty", []domain.DestinationRepository{&availabilityFlightRepository{}, &availabilityFlightRepository{}}, nil, false},
		{"no configured sources", nil, ErrFlightSourcesUnavailable, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := NewTravelService(logger.Logger{Logger: zerolog.Nop()}, &mockHotelRepository{}, tc.sources...)
			result, err := service.FindDestinations(context.Background(), 100000, 3, "MOW", travelTestDate())
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}
			if got := len(result.BestTour) > 0; got != tc.wantTour {
				t.Fatalf("expected tour=%v, got %+v", tc.wantTour, result)
			}
		})
	}
}

func TestFindDestinations_InvalidDaysDoNotCallSources(t *testing.T) {
	for _, days := range []int{0, -3} {
		source := &availabilityFlightRepository{}
		service := NewTravelService(logger.Logger{Logger: zerolog.Nop()}, &mockHotelRepository{}, source)
		if _, err := service.FindDestinations(context.Background(), 100000, days, "MOW", travelTestDate()); !errors.Is(err, ErrInvalidTravelDays) {
			t.Fatalf("days=%d: expected validation error, got %v", days, err)
		}
		if source.calls.Load() != 0 {
			t.Fatalf("days=%d: invalid input called a source", days)
		}
	}
}

func TestFindDestinations_CancellationIsNotSourceUnavailability(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := NewTravelService(logger.Logger{Logger: zerolog.Nop()}, &mockHotelRepository{},
		&availabilityFlightRepository{err: errors.New("source failed")})
	if _, err := service.FindDestinations(ctx, 100000, 3, "MOW", travelTestDate()); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected caller cancellation, got %v", err)
	}
}
