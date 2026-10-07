package cache

import (
	"context"
	"testing"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/domain"
)

func TestSharedCache_GetDeletesExpiredEntry(t *testing.T) {
	c := NewSharedCache(2, -time.Second)
	c.Set("expired", []domain.Destination{{City: "KZN"}})
	if _, ok := c.Get("expired"); ok {
		t.Fatal("expired entry was returned")
	}
	s := c.getShard("expired")
	s.mtx.Lock()
	_, exists := s.data["expired"]
	s.mtx.Unlock()
	if exists {
		t.Fatal("expired entry remained in memory after Get")
	}
}

func TestSharedCache_DeleteExpiredPreservesFreshEntries(t *testing.T) {
	c := NewSharedCache(4, time.Minute)
	c.Set("fresh", []domain.Destination{{City: "LED"}})
	s := c.getShard("expired")
	s.mtx.Lock()
	s.data["expired"] = entry{expiresAt: time.Now().Add(-time.Second)}
	s.mtx.Unlock()
	c.DeleteExpired()
	s.mtx.Lock()
	_, exists := s.data["expired"]
	s.mtx.Unlock()
	if exists {
		t.Fatal("cleanup did not delete expired entry")
	}
	if data, ok := c.Get("fresh"); !ok || len(data) != 1 || data[0].City != "LED" {
		t.Fatal("cleanup deleted a fresh entry")
	}
}

func TestSharedCache_RunCleanUpStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		NewSharedCache(2, time.Minute).RunCleanUp(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("cleanup did not stop on cancellation")
	}
}
