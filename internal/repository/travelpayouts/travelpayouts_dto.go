package travelpayouts

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
