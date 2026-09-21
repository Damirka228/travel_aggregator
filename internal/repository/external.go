package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/domain"
)

//источник, симулирующий реальный внешний API

type ExternalDestinationRepository struct {
	Delay      time.Duration
	ShouldFail bool
}

func (r ExternalDestinationRepository) GetAll(ctx context.Context) ([]domain.Destination, error) {
	select {
	case <-time.After(r.Delay):
		if r.ShouldFail {
			return nil, fmt.Errorf("external API unavailable")
		}
		return []domain.Destination{
			{City: "Пхукет", Country: "Тайланд", Price: 200000, Days: 12},
			{City: "Ереван", Country: "Армения", Price: 60000, Days: 6},
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}

}
