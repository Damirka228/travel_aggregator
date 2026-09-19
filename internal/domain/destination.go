package domain

type Destination struct {
	City    string
	Country string
	Price   float64
	Days    int
}

var destinations = []Destination{
	{"Стамбул", "Турция", 450000, 7},
	{"Бали", "Индонезия", 120000, 10},
	{"Тбилиси", "Грузия", 500000, 5},
	{"Курск", "Россия", 40000, 7},
}

func FindDestinations(budget float64, days int) []Destination {

	listDestinations := []Destination{}
	for _, v := range destinations {
		if v.Price <= budget && v.Days <= days {
			listDestinations = append(listDestinations, v)
		}
	}

	return listDestinations
}
