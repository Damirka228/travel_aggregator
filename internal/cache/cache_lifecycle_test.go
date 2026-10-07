package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"github.com/rs/zerolog"
)

type lifecycleSourceFunc func(context.Context, string) ([]domain.Destination, error)

func (f lifecycleSourceFunc) GetAll(ctx context.Context, search domain.FlightSearch) ([]domain.Destination, error) {
	return f(ctx, search.Origin)
}

type lifecycleL2 struct {
	mu    sync.Mutex
	store map[string][]domain.Destination
}

func (c *lifecycleL2) Get(_ context.Context, key string) ([]domain.Destination, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, ok := c.store[key]
	return data, ok
}

func (c *lifecycleL2) Set(_ context.Context, key string, data []domain.Destination) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = data
	return nil
}

// Done is evaluated when GetAll reaches its select, after joining singleflight.
type observedCallerContext struct {
	context.Context
	ready chan struct{}
	once  sync.Once
}

func (c *observedCallerContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.ready) })
	return c.Context.Done()
}

type lifecycleResult struct {
	data []domain.Destination
	err  error
}

func newLifecycleRepository(t *testing.T, source lifecycleSourceFunc) (*CachedDestinationRepository, *lifecycleL2) {
	t.Helper()
	l2 := &lifecycleL2{store: make(map[string][]domain.Destination)}
	repo := NewCachedDestinationRepository(
		context.Background(), source, NewSharedCache(4, time.Minute), l2,
		logger.Logger{Logger: zerolog.Nop()},
	)
	t.Cleanup(repo.Close)
	return repo, l2
}

func startLifecycleCaller(repo *CachedDestinationRepository, ctx context.Context, origin string) (<-chan lifecycleResult, <-chan struct{}) {
	observed := &observedCallerContext{Context: ctx, ready: make(chan struct{})}
	result := make(chan lifecycleResult, 1)
	go func() {
		data, err := repo.GetAll(observed, testCacheSearch(origin))
		result <- lifecycleResult{data: data, err: err}
	}()
	return result, observed.ready
}

func waitLifecycleSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for lifecycle signal")
	}
}

func receiveLifecycleResult(t *testing.T, result <-chan lifecycleResult) lifecycleResult {
	t.Helper()
	select {
	case value := <-result:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for repository result")
		return lifecycleResult{}
	}
}

func TestCachedRepository_CoalescesConcurrentRequests(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(release) }) }
	repo, _ := newLifecycleRepository(t, lifecycleSourceFunc(func(ctx context.Context, _ string) ([]domain.Destination, error) {
		calls.Add(1)
		select {
		case <-release:
			return []domain.Destination{{City: "KZN", Price: 3000}}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}))
	t.Cleanup(finish)

	results := make([]<-chan lifecycleResult, 20)
	for i := range results {
		result, ready := startLifecycleCaller(repo, context.Background(), "MOW")
		results[i] = result
		waitLifecycleSignal(t, ready)
	}
	finish()
	for _, result := range results {
		got := receiveLifecycleResult(t, result)
		if got.err != nil || len(got.data) != 1 || got.data[0].City != "KZN" {
			t.Fatalf("unexpected result: %+v", got)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("expected one shared API call, got %d", got)
	}
}

func TestCachedRepository_DifferentOriginsLoadConcurrently(t *testing.T) {
	started := make(chan string, 2)
	release := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(release) }) }
	repo, _ := newLifecycleRepository(t, lifecycleSourceFunc(func(ctx context.Context, origin string) ([]domain.Destination, error) {
		started <- origin
		select {
		case <-release:
			return []domain.Destination{{City: origin}}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}))
	t.Cleanup(finish)

	moscow, _ := startLifecycleCaller(repo, context.Background(), "MOW")
	petersburg, _ := startLifecycleCaller(repo, context.Background(), "LED")
	origins := make(map[string]bool)
	for range 2 {
		select {
		case origin := <-started:
			origins[origin] = true
		case <-time.After(5 * time.Second):
			t.Fatal("different origins did not start loading concurrently")
		}
	}
	if !origins["MOW"] || !origins["LED"] {
		t.Fatalf("unexpected origins: %v", origins)
	}
	finish()
	for origin, result := range map[string]<-chan lifecycleResult{"MOW": moscow, "LED": petersburg} {
		got := receiveLifecycleResult(t, result)
		if got.err != nil || len(got.data) != 1 || got.data[0].City != origin {
			t.Fatalf("unexpected result for %s: %+v", origin, got)
		}
	}
}

func TestCachedRepository_CallerCancellationDoesNotCancelSharedLoad(t *testing.T) {
	loadContext := make(chan context.Context, 1)
	release := make(chan struct{})
	var calls atomic.Int32
	var once sync.Once
	finish := func() { once.Do(func() { close(release) }) }
	repo, l2 := newLifecycleRepository(t, lifecycleSourceFunc(func(ctx context.Context, _ string) ([]domain.Destination, error) {
		calls.Add(1)
		loadContext <- ctx
		select {
		case <-release:
			return []domain.Destination{{City: "KZN"}}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}))
	t.Cleanup(finish)
	callerCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	first, firstReady := startLifecycleCaller(repo, callerCtx, "MOW")
	waitLifecycleSignal(t, firstReady)
	second, secondReady := startLifecycleCaller(repo, context.Background(), "MOW")
	waitLifecycleSignal(t, secondReady)

	var sharedCtx context.Context
	select {
	case sharedCtx = <-loadContext:
	case <-time.After(5 * time.Second):
		t.Fatal("shared load did not start")
	}
	cancel()
	if got := receiveLifecycleResult(t, first); !errors.Is(got.err, context.Canceled) {
		t.Fatalf("expected canceled first caller, got %v", got.err)
	}
	if err := sharedCtx.Err(); err != nil {
		t.Fatalf("first caller canceled the shared load: %v", err)
	}
	finish()
	if got := receiveLifecycleResult(t, second); got.err != nil || len(got.data) != 1 {
		t.Fatalf("remaining caller did not get data: %+v", got)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected one API call, got %d", calls.Load())
	}
	if _, ok := repo.l1.Get(testCacheKey); !ok {
		t.Fatal("result was not stored in L1")
	}
	if _, ok := l2.Get(context.Background(), testCacheKey); !ok {
		t.Fatal("result was not stored in L2")
	}
}

func TestCachedRepository_CloseWaitsAfterCallerLeaves(t *testing.T) {
	canceled := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(release) }) }
	repo, _ := newLifecycleRepository(t, lifecycleSourceFunc(func(ctx context.Context, _ string) ([]domain.Destination, error) {
		<-ctx.Done()
		close(canceled)
		<-release
		return nil, ctx.Err()
	}))
	t.Cleanup(finish)
	callerCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	result, ready := startLifecycleCaller(repo, callerCtx, "MOW")
	waitLifecycleSignal(t, ready)
	cancel()
	if got := receiveLifecycleResult(t, result); !errors.Is(got.err, context.Canceled) {
		t.Fatalf("expected caller cancellation, got %v", got.err)
	}

	closed := make(chan struct{})
	go func() {
		repo.Close()
		close(closed)
	}()
	waitLifecycleSignal(t, canceled)
	select {
	case <-closed:
		t.Fatal("Close returned before the shared load finished")
	default:
	}
	finish()
	waitLifecycleSignal(t, closed)
	if _, err := repo.GetAll(context.Background(), testCacheSearch("LED")); !errors.Is(err, context.Canceled) {
		t.Fatalf("closed repository accepted new work: %v", err)
	}
	repo.Close()
}

func TestCachedRepository_CacheHitsBalanceWaitGroup(t *testing.T) {
	for _, level := range []string{"L1", "L2"} {
		t.Run(level, func(t *testing.T) {
			var calls atomic.Int32
			repo, l2 := newLifecycleRepository(t, lifecycleSourceFunc(func(context.Context, string) ([]domain.Destination, error) {
				calls.Add(1)
				return nil, errors.New("API must not be called on cache hit")
			}))
			data := []domain.Destination{{City: "KZN"}}
			if level == "L1" {
				repo.l1.Set(testCacheKey, data)
			} else {
				_ = l2.Set(context.Background(), testCacheKey, data)
			}
			got, err := repo.GetAll(context.Background(), testCacheSearch("MOW"))
			if err != nil || len(got) != 1 {
				t.Fatalf("cache hit failed: %v, %v", got, err)
			}
			closed := make(chan struct{})
			go func() {
				repo.Close()
				close(closed)
			}()
			waitLifecycleSignal(t, closed)
			if calls.Load() != 0 {
				t.Fatal("cache hit called API")
			}
		})
	}
}

func TestCachedRepository_PreservesAPIError(t *testing.T) {
	sourceErr := errors.New("upstream unavailable")
	repo, _ := newLifecycleRepository(t, lifecycleSourceFunc(func(context.Context, string) ([]domain.Destination, error) {
		return nil, sourceErr
	}))
	if _, err := repo.GetAll(context.Background(), testCacheSearch("MOW")); !errors.Is(err, sourceErr) {
		t.Fatalf("API error was not preserved: %v", err)
	}
}

func TestCachedRepository_ApplicationCancellationStopsSharedLoad(t *testing.T) {
	appCtx, cancelApp := context.WithCancel(context.Background())
	t.Cleanup(cancelApp)
	started := make(chan struct{})
	source := lifecycleSourceFunc(func(ctx context.Context, _ string) ([]domain.Destination, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	repo := NewCachedDestinationRepository(appCtx, source, NewSharedCache(2, time.Minute),
		&lifecycleL2{store: make(map[string][]domain.Destination)}, logger.Logger{Logger: zerolog.Nop()})
	t.Cleanup(repo.Close)
	result, _ := startLifecycleCaller(repo, context.Background(), "MOW")
	waitLifecycleSignal(t, started)
	cancelApp()
	if got := receiveLifecycleResult(t, result); !errors.Is(got.err, context.Canceled) {
		t.Fatalf("application cancellation did not reach shared load: %v", got.err)
	}
}
