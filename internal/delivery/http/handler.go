package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/usecase"
)

type Handler struct {
	service *usecase.TravelService
}

func NewHandler(service *usecase.TravelService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Destinations(w http.ResponseWriter, r *http.Request) {
	budget, err := strconv.ParseFloat(r.URL.Query().Get("budget"), 64)
	if err != nil {
		http.Error(w, "invalid budget", http.StatusBadRequest)
		return
	}
	days, err := strconv.Atoi(r.URL.Query().Get("days"))
	if err != nil {
		http.Error(w, "invalid days: expected an integer", http.StatusBadRequest)
		return
	}
	origin := r.URL.Query().Get("city")
	if origin == "" {
		http.Error(w, "invalid city: origin is required", http.StatusBadRequest)
		return
	}

	departureDate, err := time.Parse(
		"2006-01-02",
		r.URL.Query().Get("departure_date"),
	)
	if err != nil {
		http.Error(
			w,
			"invalid departure_date: expected YYYY-MM-DD",
			http.StatusBadRequest,
		)
		return
	}

	result, err := h.service.FindDestinations(r.Context(), budget, days, origin, departureDate)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidBudget) {
			http.Error(w, "invalid budget: expected a finite number greater than 1000", http.StatusBadRequest)
			return
		}
		if errors.Is(err, usecase.ErrInvalidDepartureDate) {
			http.Error(w, "invalid departure_date", http.StatusBadRequest)
			return
		}
		if errors.Is(err, usecase.ErrInvalidTravelDays) {
			http.Error(
				w,
				"invalid days: expected a positive integer",
				http.StatusBadRequest,
			)
			return
		}
		if errors.Is(err, usecase.ErrFlightSourcesUnavailable) {
			http.Error(w, "flight sources temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
