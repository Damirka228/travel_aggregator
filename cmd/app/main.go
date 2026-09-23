package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	deliveryhttp "github.com/Damirka228/travel_aggregator/internal/delivery/http"
	"github.com/Damirka228/travel_aggregator/internal/repository/postgres"
	"github.com/Damirka228/travel_aggregator/internal/repository/travelpayouts"
	"github.com/Damirka228/travel_aggregator/internal/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, берем системные переменные")
	}

	token := os.Getenv("TRAVELPAYOUTS_TOKEN")

	pool, err := pgxpool.New(context.Background(), "postgres://travel:travel@localhost:5432/travel_aggregator?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	postgresRepo := postgres.NewPostgresDestinationRepository(pool)

	//repo1 := &repository.InMemoryDestinationRepository{}	//направления которые хранятся просто в мапе
	repoApi := &travelpayouts.TravelpayoutsRepository{Token: token, Client: &http.Client{Timeout: 10 * time.Second}}
	service := usecase.NewTravelService(repoApi, postgresRepo)
	handler := deliveryhttp.NewHandler(service)

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Get("/destinations", handler.Destinations)

	if err := http.ListenAndServe(":8080", router); err != nil {
		fmt.Println("error start server:", err)
		return
	}
}
