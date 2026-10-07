package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"golang.org/x/sync/singleflight"
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

	loadGroup singleflight.Group
	loadCtx   context.Context

	cancelLoads context.CancelFunc
	loadMtx     sync.Mutex
	loadWg      sync.WaitGroup
	closing     bool
}

func NewCachedDestinationRepository(appctx context.Context, service domain.DestinationRepository, l1 *SharedCache, l2 l2cache, log logger.Logger) *CachedDestinationRepository {
	loadCtx, cancelLoads := context.WithCancel(appctx)
	return &CachedDestinationRepository{
		service:     service,
		l1:          l1,
		l2:          l2,
		log:         log,
		loadCtx:     loadCtx,
		cancelLoads: cancelLoads,
	}
}

func (r *CachedDestinationRepository) GetAll(ctx context.Context, search domain.FlightSearch) ([]domain.Destination, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.loadMtx.Lock()

	if r.closing {
		r.loadMtx.Unlock()
		return nil, context.Canceled
	}

	if err := r.loadCtx.Err(); err != nil {
		r.loadMtx.Unlock()
		return nil, err
	}

	r.loadWg.Add(1)
	r.loadMtx.Unlock()

	cacheKey := fmt.Sprintf("destinations:%s:%s:%s", search.Origin, search.DepartureDate.Format("2006-01-02"), search.ReturnDate.Format("2006-01-02"))

	data, ok := r.l1.Get(cacheKey)
	if ok {
		r.log.Debug().Str("key", cacheKey).Msg("Попадание в L1 кэш (ОЗУ)")
		r.loadWg.Done()
		return data, nil
	}

	resultCh := r.loadGroup.DoChan(cacheKey, func() (any, error) {
		loadCtx, cancel := context.WithTimeout(r.loadCtx, 10*time.Second)
		defer cancel()

		data, ok := r.l1.Get(cacheKey)
		if ok {
			return data, nil
		}
		data, ok = r.l2.Get(loadCtx, cacheKey)
		if ok {
			r.log.Debug().Str("key", cacheKey).Msg("Промах L1. Попадание в L2 кэш (Redis)")
			r.l1.Set(cacheKey, data)
			return data, nil
		}
		r.log.Info().Str("origin", search.Origin).Msg("Промах всех кэшей. Запрос в API Travelpayouts...")

		data, err := r.service.GetAll(loadCtx, search)
		if err != nil {
			return nil, fmt.Errorf("api travelpayouts error, err: %w", err)
		}

		r.l1.Set(cacheKey, data)
		if err = r.l2.Set(loadCtx, cacheKey, data); err != nil {
			r.log.Error().Err(err).Str("key", cacheKey).Msg("Не удалось сохранить данные в Redis (L2)")
		}
		return data, nil
	})

	replyCh := make(chan singleflight.Result, 1)

	//ожидаем результат из общей загрузики, передаем юзеру, завершаем Done()
	go func() {
		defer r.loadWg.Done()

		result := <-resultCh
		replyCh <- result
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-replyCh:
		if result.Err != nil {
			return nil, result.Err
		}

		return result.Val.([]domain.Destination), nil
	}

}

func (r *CachedDestinationRepository) Close() {
	r.loadMtx.Lock()

	r.closing = true
	r.cancelLoads()

	r.loadMtx.Unlock()

	r.loadWg.Wait()
}
