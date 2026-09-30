package usecase

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"golang.org/x/sync/errgroup"
)

type TicketItem struct {
	domain.Destination
	Description string `json:"description"`
}

type TravelResponse struct {
	BestOption TicketItem   `json:"best_option"`
	Cheapest   []TicketItem `json:"cheapest"`
	Longest    []TicketItem `json:"longest_vacation"`
}

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

func (s *TravelService) FindDestinations(ctx context.Context, budget float64, days int, origin string) (TravelResponse, error) {
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

	filtered := []domain.Destination{}
	for _, d := range all {
		if d.Price <= budget && d.Days <= days {
			filtered = append(filtered, d)
		}
	}

	if len(filtered) == 0 {
		return TravelResponse{}, nil
	}

	//5 самых дешевых билетов
	cheapestList := append([]domain.Destination{}, filtered...)
	sort.Slice(cheapestList, func(i, j int) bool {
		return cheapestList[i].Price < cheapestList[j].Price
	})
	if len(cheapestList) > 5 {
		cheapestList = cheapestList[:5]
	}

	//5 больше дней отдыха
	longestList := append([]domain.Destination{}, filtered...)
	sort.Slice(longestList, func(i, j int) bool {
		return longestList[i].Days > longestList[j].Days
	})
	if len(longestList) > 3 {
		longestList = longestList[:3]
	}

	//самый лучший билет. Цена + (Дни * Коэффициент ценности дня). Ищем элемент с минимальным весом
	bestOption := filtered[0]
	minScore := bestOption.Price + (float64(bestOption.Days) * 1000.0)

	for _, d := range filtered {
		score := d.Price + (float64(d.Days) * 1000.0)
		if score < minScore {
			minScore = score
			bestOption = d
		}
	}

	bestCard := TicketItem{
		Destination: bestOption,
		Description: fmt.Sprintf("Самый сбалансированный перелет! Авиабилет туда-обратно в г. %s, %d дн. между рейсами всего за %.0f руб.",
			domain.GetCityNameByIATA(bestOption.City), bestOption.Days, bestOption.Price),
	}

	cheapestCards := make([]TicketItem, 0, len(cheapestList))
	for i, d := range cheapestList {
		desc := fmt.Sprintf("Дешевый перелет туда-обратно (Топ-%d)! Билет в г. %s за %.0f руб. (%d дн. между рейсами)",
			i+1, domain.GetCityNameByIATA(d.City), d.Price, d.Days)
		cheapestCards = append(cheapestCards, TicketItem{Destination: d, Description: desc})
	}

	longestCards := make([]TicketItem, 0, len(longestList))
	for i, d := range longestList {
		desc := fmt.Sprintf("Максимальное время пребывания (Топ-%d)! %d дней в г. %s между рейсами. Цена билетов туда-обратно: %.0f руб.",
			i+1, d.Days, domain.GetCityNameByIATA(d.City), d.Price)
		longestCards = append(longestCards, TicketItem{Destination: d, Description: desc})
	}

	return TravelResponse{
		BestOption: bestCard,
		Cheapest:   cheapestCards,
		Longest:    longestCards,
	}, nil
}
