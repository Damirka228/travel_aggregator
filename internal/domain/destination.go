package domain

import (
	"context"
	"time"
)

type FlightSearch struct {
	Origin        string
	DepartureDate time.Time //календарная дата пользователя!!!!
	ReturnDate    time.Time
}

type Destination struct {
	ID      int
	Origin  string
	City    string
	Country string
	Price   float64
	Days    int

	DepartureAt time.Time //дата и время конкретного вылета!!!!
	ReturnAt    time.Time
}

type DestinationRepository interface {
	GetAll(ctx context.Context, search FlightSearch) ([]Destination, error)
}
