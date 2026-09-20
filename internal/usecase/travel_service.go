package usecase

import "github.com/Damirka228/travel_aggregator/internal/domain"


type TravelService struct {
	repo domain.DestinationRepository
}

func NewTravelService (repo domain.DestinationRepository) *TravelService {
	return &TravelService{repo: repo}
}

func (s *TravelService) FindDestinations(budget float64, days int) ([]domain.Destination, error) {
    all, err := s.repo.GetAll()
    if err != nil {
        return nil, err
    }

    result := []domain.Destination{}
    for _, d := range all {
        if d.Price <= budget && d.Days <= days {
            result = append(result, d)
        }
    }
    return result, nil
}