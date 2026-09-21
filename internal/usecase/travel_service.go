package usecase

import (
	"context"
	"sync"
	"time"

	"log"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"golang.org/x/sync/errgroup"
)

type TravelService struct {
	repos []domain.DestinationRepository
}

func NewTravelService(repos ...domain.DestinationRepository) *TravelService {
	return &TravelService{repos: repos}
}

func (s *TravelService) FindDestinations(ctx context.Context, budget float64, days int) ([]domain.Destination, error) {
	var (
		mtx sync.Mutex
		all []domain.Destination
		g   errgroup.Group
	)

	for _, repo := range s.repos {
		repo := repo
		g.Go(func() error {
			reqContext, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()

			items, err := repo.GetAll(reqContext)
			if err != nil {
				log.Printf("источник %T не ответил: %v", repo, err)
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
