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

type TourItem struct {
	Flight      domain.Destination `json:"flight"`
	Hotel       domain.Hotel       `json:"hotel"`
	TotalPrice  float64            `json:"total_price"`
	Description string             `json:"description"`
}

type TravelResponse struct {
	BestTour []TourItem `json:"best_tour"`
	Cheapest []TourItem `json:"cheapest_tours"`
	Longest  []TourItem `json:"longest_tours"`
}

type TravelService struct {
	flightRepos []domain.DestinationRepository
	hotelRepo   domain.HotelRepository
	log         logger.Logger
}

func NewTravelService(log logger.Logger, hotelRepo domain.HotelRepository, flightRepos ...domain.DestinationRepository) *TravelService {
	return &TravelService{
		flightRepos: flightRepos,
		hotelRepo:   hotelRepo,
		log:         log,
	}
}

func (s *TravelService) FindDestinations(ctx context.Context, budget float64, days int, origin string) (TravelResponse, error) {
	var (
		mtx        sync.Mutex
		allFlights []domain.Destination
		g          errgroup.Group
	)

	for _, repo := range s.flightRepos {
		repo := repo
		g.Go(func() error {
			reqContext, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			items, err := repo.GetAll(reqContext, origin)
			if err != nil {
				s.log.Warn().Err(err).Str("repo_type", fmt.Sprintf("%T", repo)).Msg("Источник билетов не ответил")
				return nil
			}

			mtx.Lock()
			allFlights = append(allFlights, items...)
			mtx.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return TravelResponse{}, err
	}

	var combinedTours []TourItem

	for _, flight := range allFlights {
		hotels, err := s.hotelRepo.GetByCity(ctx, flight.City)
		if err != nil || len(hotels) == 0 {
			continue
		}

		for _, hotel := range hotels {
			totalPrice := flight.Price + (hotel.Price * float64(days))

			if totalPrice > budget || flight.Days > days {
				continue
			}

			desc := fmt.Sprintf("Выгодный тур в г. %s! Перелет туда-обратно + отель %s (%d) на %d дней между рейсами. Всё включено!",
				domain.GetCityNameByIATA(flight.City), hotel.Name, hotel.Stars, days)

			combinedTours = append(combinedTours, TourItem{
				Flight:      flight,
				Hotel:       hotel,
				TotalPrice:  totalPrice,
				Description: desc,
			})
		}
	}

	if len(combinedTours) == 0 {
		return TravelResponse{}, nil
	}

	cheapestTours := append([]TourItem{}, combinedTours...)
	sort.Slice(cheapestTours, func(i, j int) bool {
		return cheapestTours[i].TotalPrice < cheapestTours[j].TotalPrice
	})
	if len(cheapestTours) > 5 {
		cheapestTours = cheapestTours[:5]
	}

	luxuryTours := append([]TourItem{}, combinedTours...)
	sort.Slice(luxuryTours, func(i, j int) bool {
		return luxuryTours[i].Hotel.Stars > luxuryTours[j].Hotel.Stars
	})
	if len(luxuryTours) > 5 {
		luxuryTours = luxuryTours[:5]
	}

	bestTour := combinedTours[0]
	minScore := bestTour.TotalPrice - (float64(bestTour.Hotel.Stars) * 2000.0)

	for _, tItem := range combinedTours {
		score := tItem.TotalPrice - (float64(tItem.Hotel.Stars) * 2000.0)
		if score < minScore {
			minScore = score
			bestTour = tItem
		}
	}

	bestTour.Description = fmt.Sprintf("РЕКОМЕНДУЕМЫЙ ЛУЧШИЙ ТУР! Идеальный перелет и отель %s (%d) в г. %s на %d дней. Общая цена за ВСЁ: %.0f руб.",
		bestTour.Hotel.Name, bestTour.Hotel.Stars, domain.GetCityNameByIATA(bestTour.Flight.City), days, bestTour.TotalPrice)

	return TravelResponse{
		BestTour: []TourItem{bestTour},
		Cheapest: cheapestTours,
		Longest:  luxuryTours,
	}, nil
}
