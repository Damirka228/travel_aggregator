package cache

import (
	"context"
	"fmt"
	"log"

	"github.com/Damirka228/travel_aggregator/internal/domain"
)

type l2cache interface {
	Get(ctx context.Context, key string) ([]domain.Destination, bool)
	Set(ctx context.Context, key string, val []domain.Destination) error
}

type CachedDestinationRepository struct {
	service domain.DestinationRepository
	l1      *SharedCache
	l2      l2cache
}

func NewCachedDestinationRepository(service domain.DestinationRepository, l1 *SharedCache, l2 l2cache) *CachedDestinationRepository {
	return &CachedDestinationRepository{
		service: service,
		l1:      l1,
		l2:      l2,
	}
}

func (r *CachedDestinationRepository) GetAll(ctx context.Context) ([]domain.Destination, error) {
	cacheKey := "destinations:MOW"

	data, ok := r.l1.Get(cacheKey)
	if ok {
		return data, nil
	}

	data, ok = r.l2.Get(ctx, cacheKey)
	if ok {
		r.l1.Set(cacheKey, data)
		return data, nil
	}

	data, err := r.service.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("api travelpayouts error, err: %s", err)
	}

	r.l1.Set(cacheKey, data)

	if err = r.l2.Set(ctx, cacheKey, data); err != nil {
		log.Printf("error set data redis, err: %s", err)
	}
	return data, nil
}
