package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/cache"
	deliveryhttp "github.com/Damirka228/travel_aggregator/internal/delivery/http"
	"github.com/Damirka228/travel_aggregator/internal/repository/postgres"
	"github.com/Damirka228/travel_aggregator/internal/repository/redis"
	"github.com/Damirka228/travel_aggregator/internal/repository/travelpayouts"
	"github.com/Damirka228/travel_aggregator/internal/usecase"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type App struct {
	httpServer *http.Server
	pgPool     *pgxpool.Pool
}

func NewApp() *App {
	ctx := context.Background()
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден")
	}

	token := os.Getenv("TRAVELPAYOUTS_TOKEN")
	postgreconnStr := os.Getenv("POSTGRESQL_STR")

	pool, err := pgxpool.New(ctx, postgreconnStr)
	if err != nil {
		log.Fatal(err)
	}

	cacheL1Repo := cache.NewSharedCache(256, 5*time.Minute)
	clientRedis, err := redis.NewClient(ctx, "localhost:6379")
	if err != nil {
		log.Fatalf("error connect redis, err: %s", err)
	}
	cacheRedisL2Repo := redis.NewDestinationsCache(clientRedis, 1*time.Hour)

	postgresRepo := postgres.NewPostgresDestinationRepository(pool)
	repoApi := travelpayouts.NewTravelpayoutsRepository(token)
	cachedApiRepo := cache.NewCachedDestinationRepository(repoApi, cacheL1Repo, cacheRedisL2Repo)

	//jwt
	postgresUserRepo := postgres.NewPostgresUserRepository(pool)
	authService := usecase.NewAuthService(postgresUserRepo)
	authHandler := deliveryhttp.NewAuthHandler(authService)

	service := usecase.NewTravelService(postgresRepo, cachedApiRepo)
	handler := deliveryhttp.NewHandler(service)

	router := chi.NewRouter()
	router.Use(middleware.Logger)

	router.Get("/destinations", handler.Destinations)
	router.Post("/auth/signup", authHandler.SignUp)
	router.Post("/auth/signin", authHandler.SignIn)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	return &App{
		httpServer: server,
	}
}

func (a *App) Run() error {
	fmt.Println("statr server")
	return a.httpServer.ListenAndServe()
}

func (a *App) Shutdown() {
	a.pgPool.Close()
}
