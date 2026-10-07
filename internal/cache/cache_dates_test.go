package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"github.com/rs/zerolog"
)

const testCacheKey = "destinations:MOW:2026-11-10:2026-11-17"

func testCacheSearch(origin string) domain.FlightSearch {
	return domain.FlightSearch{
		Origin:        origin,
		DepartureDate: time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC),
		ReturnDate:    time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC),
	}
}

type cacheDateSource func(context.Context, domain.FlightSearch) ([]domain.Destination, error)

func (f cacheDateSource) GetAll(ctx context.Context, search domain.FlightSearch) ([]domain.Destination, error) {
	return f(ctx, search)
}

func newDateCacheRepository(t *testing.T, source cacheDateSource, l2 *lifecycleL2) *CachedDestinationRepository {
	t.Helper()
	repo := NewCachedDestinationRepository(context.Background(), source, NewSharedCache(4, time.Minute), l2, logger.Logger{Logger: zerolog.Nop()})
	t.Cleanup(repo.Close)
	return repo
}

func TestCachedRepository_DatesSeparateCacheEntries(t *testing.T) {
	for _, field := range []string{"departure", "return"} {
		for _, level := range []string{"L1", "L2"} {
			t.Run(field+"/"+level, func(t *testing.T) {
				first := testCacheSearch("MOW")
				second := first
				if field == "departure" {
					second.DepartureDate = first.DepartureDate.AddDate(0, 0, 1)
				} else {
					second.ReturnDate = first.ReturnDate.AddDate(0, 0, 1)
				}
				var calls atomic.Int32
				source := cacheDateSource(func(_ context.Context, search domain.FlightSearch) ([]domain.Destination, error) {
					price := float64(calls.Add(1)) * 1000
					return []domain.Destination{{Origin: search.Origin, City: "KZN", Price: price, DepartureAt: search.DepartureDate, ReturnAt: search.ReturnDate}}, nil
				})
				l2 := &lifecycleL2{store: make(map[string][]domain.Destination)}
				repo := newDateCacheRepository(t, source, l2)
				for _, search := range []domain.FlightSearch{first, second} {
					if _, err := repo.GetAll(context.Background(), search); err != nil {
						t.Fatal(err)
					}
				}
				if level == "L2" {
					repo.Close()
					repo = newDateCacheRepository(t, source, l2)
				}
				for i, search := range []domain.FlightSearch{first, second, first, second} {
					got, err := repo.GetAll(context.Background(), search)
					wantPrice := float64(i%2+1) * 1000
					if err != nil || len(got) != 1 || got[0].Price != wantPrice || !got[0].DepartureAt.Equal(search.DepartureDate) || !got[0].ReturnAt.Equal(search.ReturnDate) {
						t.Fatalf("cache mixed dates: search=%+v data=%+v err=%v", search, got, err)
					}
				}
				if calls.Load() != 2 {
					t.Fatalf("expected two loads then cache hits, got %d", calls.Load())
				}
			})
		}
	}
}

func TestCachedRepository_DifferentDatesLoadConcurrently(t *testing.T) {
	for _, field := range []string{"departure", "return"} {
		t.Run(field, func(t *testing.T) {
			first := testCacheSearch("MOW")
			second := first
			if field == "departure" {
				second.DepartureDate = first.DepartureDate.AddDate(0, 0, 1)
			} else {
				second.ReturnDate = first.ReturnDate.AddDate(0, 0, 1)
			}
			started := make(chan domain.FlightSearch, 2)
			release := make(chan struct{})
			var once sync.Once
			finish := func() { once.Do(func() { close(release) }) }
			source := cacheDateSource(func(ctx context.Context, search domain.FlightSearch) ([]domain.Destination, error) {
				started <- search
				select {
				case <-release:
					return []domain.Destination{{DepartureAt: search.DepartureDate, ReturnAt: search.ReturnDate}}, nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			})
			repo := newDateCacheRepository(t, source, &lifecycleL2{store: make(map[string][]domain.Destination)})
			t.Cleanup(finish)
			results := make([]chan lifecycleResult, 2)
			for i, search := range []domain.FlightSearch{first, second} {
				results[i] = make(chan lifecycleResult, 1)
				go func(search domain.FlightSearch, reply chan lifecycleResult) {
					data, err := repo.GetAll(context.Background(), search)
					reply <- lifecycleResult{data: data, err: err}
				}(search, results[i])
			}
			seen := make(map[domain.FlightSearch]bool)
			for range 2 {
				select {
				case search := <-started:
					seen[search] = true
				case <-time.After(5 * time.Second):
					t.Fatal("different dates were incorrectly merged into one load")
				}
			}
			if !seen[first] || !seen[second] {
				t.Fatalf("wrong searches reached source: %+v", seen)
			}
			finish()
			for i, search := range []domain.FlightSearch{first, second} {
				got := receiveLifecycleResult(t, results[i])
				if got.err != nil || len(got.data) != 1 || !got.data[0].DepartureAt.Equal(search.DepartureDate) || !got.data[0].ReturnAt.Equal(search.ReturnDate) {
					t.Fatalf("unexpected result for %+v: %+v", search, got)
				}
			}
		})
	}
}
