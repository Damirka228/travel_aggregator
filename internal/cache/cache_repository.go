package cache

import (
	"context"
	"fmt"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
)

type l2cache interface {
	Get(ctx context.Context, key string) ([]domain.Destination, bool)
	Set(ctx context.Context, key string, val []domain.Destination) error
}

type CachedDestinationRepository struct {
	service domain.DestinationRepository
	l1      *SharedCache
	l2      l2cache
	log     logger.Logger
}

func NewCachedDestinationRepository(service domain.DestinationRepository, l1 *SharedCache, l2 l2cache, log logger.Logger) *CachedDestinationRepository {
	return &CachedDestinationRepository{
		service: service,
		l1:      l1,
		l2:      l2,
		log:     log,
	}
}

func (r *CachedDestinationRepository) GetAll(ctx context.Context, origin string) ([]domain.Destination, error) {
	cacheKey := fmt.Sprintf("destinations:%s", origin)

	data, ok := r.l1.Get(cacheKey)
	if ok {
		r.log.Debug().Str("key", cacheKey).Msg("Попадание в L1 кэш (ОЗУ)")
		return data, nil
	}

	data, ok = r.l2.Get(ctx, cacheKey)
	if ok {
		r.log.Debug().Str("key", cacheKey).Msg("Промах L1. Попадание в L2 кэш (Redis)")
		r.l1.Set(cacheKey, data)
		return data, nil
	}

	r.log.Info().Str("origin", origin).Msg("Промах всех кэшей. Запрос в API Travelpayouts...")

	data, err := r.service.GetAll(ctx, origin)
	if err != nil {
		return nil, fmt.Errorf("api travelpayouts error, err: %s", err)
	}

	r.l1.Set(cacheKey, data)

	if err = r.l2.Set(ctx, cacheKey, data); err != nil {
		r.log.Error().Err(err).Str("key", cacheKey).Msg("Не удалось сохранить данные в Redis (L2)")
	}
	return data, nil
}
