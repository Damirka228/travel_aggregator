package usecase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"golang.org/x/sync/errgroup"
)

type TravelService struct {
	repos []domain.DestinationRepository
	log   logger.Logger
}

func NewTravelService(log logger.Logger, repos ...domain.DestinationRepository) *TravelService {
	return &TravelService{
		repos: repos,
		log:   log,
	}
}

func (s *TravelService) FindDestinations(ctx context.Context, budget float64, days int, origin string) ([]domain.Destination, error) {
	var (
		mtx sync.Mutex
		all []domain.Destination
		g   errgroup.Group
	)

	for _, repo := range s.repos {
		repo := repo
		g.Go(func() error {
			reqContext, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			items, err := repo.GetAll(reqContext, origin)
			if err != nil {
				s.log.Warn().Err(err).Str("repo_type", fmt.Sprintf("%T", repo)).Str("origin", origin).Msg("Один из источников билетов не ответил на запрос")
				return nil
			}

			mtx.Lock()
			all = append(all, items...)
			mtx.Unlock()
			return nil
		})
	}
	g.Wait()

	result := []domain.Destination{}
	for _, d := range all {
		if d.Price <= budget && d.Days <= days {
			result = append(result, d)
		}
	}
	return result, nil
}
