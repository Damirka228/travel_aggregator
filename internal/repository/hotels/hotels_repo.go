package hotels

import (
	"context"

	"github.com/Damirka228/travel_aggregator/internal/domain"
)

type HotelRepository struct {
}

func NewHotelRepository() *HotelRepository {
	return &HotelRepository{}
}

func (r *HotelRepository) GetByCity(ctx context.Context, city string) ([]domain.Hotel, error) {
	allHotels := map[string][]domain.Hotel{
		"KZN": {
			{ID: 201, Name: "Kazan Palace by TASIGO", City: "KZN", Price: 8500, Stars: 5},
			{ID: 202, Name: "Отель Ривьера (с аквапарком)", City: "KZN", Price: 6200, Stars: 4},
			{ID: 203, Name: "Хостел Кремлин", City: "KZN", Price: 1200, Stars: 2},
		},
		"DXB": {
			{ID: 301, Name: "Burj Al Arab (Парус)", City: "DXB", Price: 95000, Stars: 5},
			{ID: 302, Name: "Atlantis The Palm", City: "DXB", Price: 45000, Stars: 5},
			{ID: 303, Name: "Rove Downtown Dubai", City: "DXB", Price: 7000, Stars: 3},
		},
		"LED": {
			{ID: 401, Name: "Гранд Отель Европа", City: "LED", Price: 15000, Stars: 5},
			{ID: 402, Name: "Отель Астория", City: "LED", Price: 18000, Stars: 5},
			{ID: 403, Name: "Питер-Инн Хостел", City: "LED", Price: 1500, Stars: 2},
		},
		"KUF": {
			{ID: 501, Name: "Lotte Hotel Samara", City: "KUF", Price: 9000, Stars: 5},
			{ID: 502, Name: "7 Avenue Hotel & SPA", City: "KUF", Price: 5500, Stars: 5},
		},
	}

	if hotels, ok := allHotels[city]; ok {
		return hotels, nil
	}

	return []domain.Hotel{
		{ID: 999, Name: "Standard Central Hotel", City: city, Price: 3500, Stars: 3},
	}, nil
}
