package domain

import "context"

type Destination struct {
    ID      int
    City    string
    Country string
    Price   float64
    Days    int
}

type DestinationRepository interface{
	GetAll(ctx context.Context)([]Destination, error)
}

