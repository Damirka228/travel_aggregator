package domain

import "testing"

func TestGetCityNameByIATA(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"MOW", "Москва"},
		{"LED", "Санкт-Петербург"},
		{"UNKNOWN", "UNKNOWN"},
	}

	for _, tt := range tests {
		result := GetCityNameByIATA(tt.code)
		if result != tt.expected {
			t.Errorf("For %s expected %s, got %s", tt.code, tt.expected, result)
		}
	}
}
