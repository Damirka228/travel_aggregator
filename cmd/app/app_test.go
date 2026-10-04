package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

func TestShutdownWaitsForActiveRequest(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	finishRequest := func() { releaseOnce.Do(func() { close(release) }) }

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		_, _ = io.WriteString(w, "completed")
	}))
	t.Cleanup(server.Close)
	t.Cleanup(finishRequest)

	app := &App{httpServer: server.Config, log: logger.Logger{Logger: zerolog.Nop()}}
	requestResult := make(chan error, 1)
	go func() {
		resp, err := server.Client().Get(server.URL)
		if err == nil {
			defer resp.Body.Close()
			body, readErr := io.ReadAll(resp.Body)
			err = readErr
			if err == nil && string(body) != "completed" {
				err = errors.New("active request did not finish its response")
			}
		}
		requestResult <- err
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not start")
	}

	shutdownStarted := make(chan struct{})
	server.Config.RegisterOnShutdown(func() { close(shutdownStarted) })
	shutdownResult := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() { shutdownResult <- app.Shutdown(ctx) }()

	select {
	case <-shutdownStarted:
	case <-ctx.Done():
		t.Fatal("shutdown did not start")
	}
	select {
	case err := <-shutdownResult:
		t.Fatalf("shutdown returned before the active request finished: %v", err)
	default:
	}

	finishRequest()
	select {
	case err := <-requestResult:
		if err != nil {
			t.Fatalf("active request: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("active request did not finish")
	}
	if err := <-shutdownResult; err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestShutdownForcesCloseAfterDeadline(t *testing.T) {
	started := make(chan struct{})
	canceled := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(canceled)
	}))
	t.Cleanup(server.Close)
	app := &App{httpServer: server.Config, log: logger.Logger{Logger: zerolog.Nop()}}

	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		resp, err := server.Client().Get(server.URL)
		if err == nil {
			resp.Body.Close()
		}
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not start")
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	err := app.Shutdown(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline error, got %v", err)
	}
	select {
	case <-canceled:
	case <-time.After(2 * time.Second):
		t.Fatal("forced close did not cancel the active request")
	}
	select {
	case <-requestDone:
	case <-time.After(2 * time.Second):
		t.Fatal("client connection was not closed")
	}
	if err := app.Shutdown(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("repeated shutdown lost the original error: %v", err)
	}
}

func TestShutdownClosesRedisOnlyOnce(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	app := &App{redisClient: client, log: logger.Logger{Logger: zerolog.Nop()}}
	for range 2 {
		if err := app.Shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown: %v", err)
		}
	}
	if err := client.Ping(context.Background()).Err(); !errors.Is(err, redis.ErrClosed) {
		t.Fatalf("expected a closed Redis client, got %v", err)
	}
}

func TestRunReturnsListenerError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	app := &App{
		httpServer: &http.Server{Addr: listener.Addr().String()},
		log:        logger.Logger{Logger: zerolog.Nop()},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.Run(ctx); err == nil {
		t.Fatal("expected an error when the port is already occupied")
	}
	if err := app.Shutdown(ctx); err != nil {
		t.Fatalf("cleanup after failed start: %v", err)
	}
}

func TestRunCanBeCanceledBeforeServerStarts(t *testing.T) {
	app := &App{
		httpServer: &http.Server{Addr: "127.0.0.1:0"},
		log:        logger.Logger{Logger: zerolog.Nop()},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := app.Run(ctx); err != nil {
		t.Fatalf("run with canceled context: %v", err)
	}
	shutdownCtx, stop := context.WithTimeout(context.Background(), 2*time.Second)
	defer stop()
	if err := app.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown before startup completed: %v", err)
	}
}
