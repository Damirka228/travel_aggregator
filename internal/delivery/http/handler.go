package http

import (
    "encoding/json"
    "net/http"
    "strconv"

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
        http.Error(w, "invalid days", http.StatusBadRequest)
        return
    }

    result, err := h.service.FindDestinations(budget, days)
    if err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}