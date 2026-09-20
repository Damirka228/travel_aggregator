package domain

type Destination struct {
	City    string
	Country string
	Price   float64
	Days    int
}

type DestinationRepository interface{
	GetAll()([]Destination, error)
}

