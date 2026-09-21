package main

import (
	"fmt"
	"net/http"
	"time"

	deliveryhttp "github.com/Damirka228/travel_aggregator/internal/delivery/http"
	"github.com/Damirka228/travel_aggregator/internal/repository"
	"github.com/Damirka228/travel_aggregator/internal/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	repo1 := &repository.InMemoryDestinationRepository{}
	repo2 := &repository.ExternalDestinationRepository{Delay: 5 * time.Second}
	service := usecase.NewTravelService(repo1, repo2)
	handler := deliveryhttp.NewHandler(service)

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Get("/destinations", handler.Destinations)

	if err := http.ListenAndServe(":8080", router); err != nil {
		fmt.Println("error start server:", err)
		return
	}
}
