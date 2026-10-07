package domain

type priceInfo struct {
	Price       float64 `json:"price"`
	Airline     string  `json:"airline"`
	DepartureAt string  `json:"departure_at"`
	ReturnAt    string  `json:"return_at"`
}

type CheapPriceResponse struct {
	Success bool                            `json:"success"`
	Data    map[string]map[string]priceInfo `json:"data"`
}

type FlightPrice struct {
	Origin      string  `json:"origin"`
	Destination string  `json:"destination"`
	Price       float64 `json:"price"`
	DepartureAt string  `json:"departure_at"`
	ReturnAt    string  `json:"return_at"`
}

type PricesForDatesResponse struct {
	Success bool          `json:"success"`
	Data    []FlightPrice `json:"data"`
}
