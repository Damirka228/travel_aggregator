package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindDestinations(t *testing.T) {
	tests := []struct {
		name   string
		budget float64
		days   int
		want   []Destination
	}{
		{
			name:   "хватает на все по бюджету",
			budget: 600000,
			days:   15,
			want: []Destination{
				{"Стамбул", "Турция", 450000, 7},
				{"Бали", "Индонезия", 120000, 10},
				{"Тбилиси", "Грузия", 500000, 5},
				{"Курск", "Россия", 40000, 7},
			},
		},
		{
			name:   "ожидаем Бали и Курск, но не Стамбул и Тбилиси",
			budget: 150000,
			days:   10,
			want: []Destination{
				{"Бали", "Индонезия", 120000, 10},
				{"Курск", "Россия", 40000, 7},
			},
		},
		{
			name:   "никуда не едем сидим дома",
			budget: 0,
			days:   0,
			want:   []Destination{},
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			res := FindDestinations(c.budget, c.days)
			assert.Equal(t, c.want, res)
		})
	}
}
