package main

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/cache"
	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type shutdownFlightSource struct {
	started  chan struct{}
	canceled chan struct{}
	release  chan struct{}
}

func (s *shutdownFlightSource) GetAll(ctx context.Context, _ domain.FlightSearch) ([]domain.Destination, error) {
	close(s.started)
	<-ctx.Done()
	close(s.canceled)
	<-s.release
	return nil, ctx.Err()
}

type shutdownL2Stub struct{}

func (shutdownL2Stub) Get(context.Context, string) ([]domain.Destination, bool) {
	return nil, false
}

func (shutdownL2Stub) Set(context.Context, string, []domain.Destination) error {
	return nil
}

func awaitShutdownSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for shutdown signal")
	}
}

func TestShutdownWaitsForSharedLoadBeforeClosingRedis(t *testing.T) {
	source := &shutdownFlightSource{started: make(chan struct{}), canceled: make(chan struct{}), release: make(chan struct{})}
	var once sync.Once
	finish := func() { once.Do(func() { close(source.release) }) }
	log := logger.Logger{Logger: zerolog.Nop()}
	repo := cache.NewCachedDestinationRepository(context.Background(), source, cache.NewSharedCache(2, time.Minute), shutdownL2Stub{}, log)
	dialErr := errors.New("test dialer: Redis server not required")
	client := redis.NewClient(&redis.Options{
		Addr:       "127.0.0.1:1",
		MaxRetries: -1,
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			return nil, dialErr
		},
	})
	app := &App{cachedAPIRepo: repo, redisClient: client, log: log}
	t.Cleanup(func() { _ = app.Shutdown(context.Background()) })
	t.Cleanup(finish)
	callerCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	requestResult := make(chan error, 1)
	go func() {
		_, err := repo.GetAll(callerCtx, domain.FlightSearch{
			Origin:        "MOW",
			DepartureDate: time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC),
			ReturnDate:    time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC),
		})
		requestResult <- err
	}()
	awaitShutdownSignal(t, source.started)
	cancel()
	select {
	case err := <-requestResult:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected caller cancellation, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("caller did not stop waiting")
	}

	shutdownResult := make(chan error, 1)
	go func() { shutdownResult <- app.Shutdown(context.Background()) }()
	awaitShutdownSignal(t, source.canceled)
	select {
	case err := <-shutdownResult:
		t.Fatalf("shutdown returned before shared load finished: %v", err)
	default:
	}
	if err := client.Ping(context.Background()).Err(); !errors.Is(err, dialErr) {
		t.Fatalf("Redis client closed before shared load finished: %v", err)
	}
	finish()
	select {
	case err := <-shutdownResult:
		if err != nil {
			t.Fatalf("shutdown failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown did not finish after shared load")
	}
	if err := client.Ping(context.Background()).Err(); !errors.Is(err, redis.ErrClosed) {
		t.Fatalf("Redis client was not closed after shared load: %v", err)
	}
}
