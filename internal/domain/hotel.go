package domain

import "context"

type Hotel struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`  
	City  string  `json:"city"` 
	Price float64 `json:"price"` 
	Stars int     `json:"stars"`
}

type HotelRepository interface {
	GetByCity (ctx context.Context, city string) ([]Hotel, error)
}