package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"github.com/rs/zerolog"
)

func travelTestDate() time.Time {
	return time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
}

type dateFlightSource func(context.Context, domain.FlightSearch) ([]domain.Destination, error)

func (f dateFlightSource) GetAll(ctx context.Context, search domain.FlightSearch) ([]domain.Destination, error) {
	return f(ctx, search)
}

type dateHotelSource func(context.Context, string) ([]domain.Hotel, error)

func (f dateHotelSource) GetByCity(ctx context.Context, city string) ([]domain.Hotel, error) {
	return f(ctx, city)
}

func TestFindDestinations_ForwardsCalendarDatesToAllSources(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		input                time.Time
		days                 int
		departure, returning string
	}{
		{"regular week", travelTestDate(), 7, "2026-11-10", "2026-11-17"},
		{"month boundary", time.Date(2026, 11, 28, 0, 0, 0, 0, time.UTC), 7, "2026-11-28", "2026-12-05"},
		{"leap year", time.Date(2028, 2, 28, 0, 0, 0, 0, time.UTC), 2, "2028-02-28", "2028-03-01"},
		{"year boundary", time.Date(2026, 12, 30, 0, 0, 0, 0, time.UTC), 3, "2026-12-30", "2027-01-02"},
		{"time and offset removed", time.Date(2026, 11, 10, 23, 30, 0, 0, time.FixedZone("UTC+3", 3*3600)), 7, "2026-11-10", "2026-11-17"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			seen := make(chan domain.FlightSearch, 2)
			source := dateFlightSource(func(_ context.Context, search domain.FlightSearch) ([]domain.Destination, error) {
				seen <- search
				return nil, nil
			})
			service := NewTravelService(logger.Logger{Logger: zerolog.Nop()}, &mockHotelRepository{}, source, source)
			if _, err := service.FindDestinations(context.Background(), 100000, tc.days, "MOW", tc.input); err != nil {
				t.Fatal(err)
			}
			for range 2 {
				select {
				case search := <-seen:
					wantDeparture, _ := time.Parse("2006-01-02", tc.departure)
					wantReturn, _ := time.Parse("2006-01-02", tc.returning)
					if search.Origin != "MOW" || !search.DepartureDate.Equal(wantDeparture) || !search.ReturnDate.Equal(wantReturn) {
						t.Fatalf("unexpected search: %+v", search)
					}
					if search.DepartureDate.Location() != time.UTC || search.ReturnDate.Location() != time.UTC {
						t.Fatal("calendar dates must use UTC")
					}
				default:
					t.Fatal("a flight source did not receive the search")
				}
			}
		})
	}
}

func TestFindDestinations_MissingDateDoesNotCallSources(t *testing.T) {
	source := &availabilityFlightRepository{}
	service := NewTravelService(logger.Logger{Logger: zerolog.Nop()}, &mockHotelRepository{}, source)
	_, err := service.FindDestinations(context.Background(), 100000, 7, "MOW", time.Time{})
	if !errors.Is(err, ErrInvalidDepartureDate) || source.calls.Load() != 0 {
		t.Fatalf("expected date validation before source calls: err=%v calls=%d", err, source.calls.Load())
	}
}

func TestFindDestinations_ExactDurationAndBudget(t *testing.T) {
	for _, tc := range []struct {
		name      string
		budget    float64
		wantTours int
	}{
		{"within budget", 10000, 1},
		{"exact budget", 8500, 1},
		{"over budget", 8499, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flights := &testFlightRepository{flights: []domain.Destination{
				{ID: 1, City: "SHORT", Price: 5000, Days: 5},
				{ID: 2, City: "EXACT", Price: 5000, Days: 7},
				{ID: 3, City: "LONG", Price: 5000, Days: 9},
			}}
			var hotelCities []string
			hotels := dateHotelSource(func(_ context.Context, city string) ([]domain.Hotel, error) {
				hotelCities = append(hotelCities, city)
				return []domain.Hotel{{Name: "Test hotel", City: city, Price: 500, Stars: 3}}, nil
			})
			service := NewTravelService(logger.Logger{Logger: zerolog.Nop()}, hotels, flights)
			result, err := service.FindDestinations(context.Background(), tc.budget, 7, "MOW", travelTestDate())
			if err != nil {
				t.Fatal(err)
			}
			if len(hotelCities) != 1 || hotelCities[0] != "EXACT" {
				t.Fatalf("hotels fetched for wrong duration: %v", hotelCities)
			}
			for _, tours := range [][]TourItem{result.BestTour, result.Cheapest, result.Longest} {
				if len(tours) != tc.wantTours {
					t.Fatalf("unexpected tours: %+v", tours)
				}
				for _, tour := range tours {
					if tour.Flight.ID != 2 || tour.Flight.Days != 7 || tour.TotalPrice != 8500 {
						t.Fatalf("expected 5000 + 500*7 = 8500 for exact duration: %+v", tour)
					}
				}
			}
		})
	}
}

func TestFindDestinations_HotelErrorSkipsFlight(t *testing.T) {
	flights := &testFlightRepository{flights: []domain.Destination{{City: "KZN", Price: 5000, Days: 7}}}
	hotels := dateHotelSource(func(context.Context, string) ([]domain.Hotel, error) {
		return []domain.Hotel{{Price: 500}}, errors.New("hotel source failed")
	})
	service := NewTravelService(logger.Logger{Logger: zerolog.Nop()}, hotels, flights)
	result, err := service.FindDestinations(context.Background(), 100000, 7, "MOW", travelTestDate())
	if err != nil || len(result.BestTour) != 0 || len(result.Cheapest) != 0 || len(result.Longest) != 0 {
		t.Fatalf("hotel error must skip the flight even when partial data is returned: %+v, %v", result, err)
	}
}
