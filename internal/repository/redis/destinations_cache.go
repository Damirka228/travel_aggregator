package redis

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/redis/go-redis/v9"
)

type DestinationsCache struct {
	redisClient *redis.Client
	ttl         time.Duration
}

func NewDestinationsCache(client *redis.Client, ttl time.Duration) *DestinationsCache {
	return &DestinationsCache{
		redisClient: client,
		ttl:         ttl,
	}
}

func (c *DestinationsCache) Get(ctx context.Context, key string) ([]domain.Destination, bool) {
	raw, err := c.redisClient.Get(ctx, key).Result()

	if errors.Is(err, redis.Nil) {
		return nil, false
	}

	if err != nil {
		log.Printf("error get redis: %s", err)
		return nil, false
	}
	var result []domain.Destination
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, false
	}
	return result, true

}

func (c DestinationsCache) Set(ctx context.Context, key string, val []domain.Destination) error {
	raw, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return c.redisClient.Set(ctx, key, raw, c.ttl).Err()
}
