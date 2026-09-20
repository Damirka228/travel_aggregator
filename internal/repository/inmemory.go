package repository

import "github.com/Damirka228/travel_aggregator/internal/domain"

type InMemoryDestinationRepository struct{}

func NewInMemoryDestinationRepository() *InMemoryDestinationRepository {
	return &InMemoryDestinationRepository{}
}

func (r *InMemoryDestinationRepository) GetAll() ([]domain.Destination, error) {
	return []domain.Destination{
		{City: "Стамбул", Country: "Турция", Price: 450000, Days: 7},
		{City: "Бали", Country: "Индонезия", Price: 120000, Days: 10},
		{City: "Тбилиси", Country: "Грузия", Price: 500000, Days: 5},
		{City: "Курск", Country: "Россия", Price: 40000, Days: 7},
	}, nil
}
