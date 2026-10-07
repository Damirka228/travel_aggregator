package cache

import (
	"context"
	"hash/fnv"
	"sync"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/domain"
)

type entry struct {
	data      []domain.Destination
	expiresAt time.Time
}

type shard struct {
	mtx  sync.RWMutex
	data map[string]entry
}

type SharedCache struct {
	shards []*shard
	size   uint32
	ttl    time.Duration
}

func NewSharedCache(shardCount int, ttl time.Duration) *SharedCache {
	shards := make([]*shard, shardCount)
	for val := range shards {
		shards[val] = &shard{data: make(map[string]entry)}
	}
	return &SharedCache{
		shards: shards,
		size:   uint32(shardCount),
		ttl:    ttl,
	}
}

func (s *SharedCache) getShard(key string) *shard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return s.shards[h.Sum32()%s.size]
}

func (c *SharedCache) Get(key string) ([]domain.Destination, bool) {
	s := c.getShard(key)

	s.mtx.Lock()
	defer s.mtx.Unlock()

	e, ok := s.data[key]
	if !ok {
		return nil, false
	}

	if !time.Now().Before(e.expiresAt) {
		delete(s.data, key)
		return nil, false
	}

	return e.data, true
}

func (c *SharedCache) DeleteExpired() {
	for _, s := range c.shards {
		s.mtx.Lock()

		now := time.Now()

		for key, e := range s.data {
			if !now.Before(e.expiresAt) {
				delete(s.data, key)
			}
		}
		s.mtx.Unlock()
	}
}

func (c *SharedCache) RunCleanUp(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.DeleteExpired()
		}
	}
}

func (c *SharedCache) Set(key string, data []domain.Destination) {
	s := c.getShard(key)

	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.data[key] = entry{
		data:      data,
		expiresAt: time.Now().Add(c.ttl),
	}
}
