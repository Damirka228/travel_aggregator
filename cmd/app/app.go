package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/cache"
	deliveryhttp "github.com/Damirka228/travel_aggregator/internal/delivery/http"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
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
	log        logger.Logger
}

func NewApp() *App {
	appLogger := logger.New()
	appLogger.Info().Msg("the logger is initialized")

	ctx := context.Background()
	if err := godotenv.Load(); err != nil {
		appLogger.Warn().Msg("Файл .env не найден")
	}

	token := os.Getenv("TRAVELPAYOUTS_TOKEN")
	postgreconnStr := os.Getenv("POSTGRESQL_STR")

	pool, err := pgxpool.New(ctx, postgreconnStr)
	if err != nil {
		appLogger.Fatal().Err(err).Msg("Критический сбой: не удалось подключить пул Postgres")
	}

	cacheL1Repo := cache.NewSharedCache(256, 5*time.Minute)
	clientRedis, err := redis.NewClient(ctx, "localhost:6379")
	if err != nil {
		appLogger.Fatal().Err(err).Msg("Критический сбой: не удалось подключиться к Redis")
	}
	cacheRedisL2Repo := redis.NewDestinationsCache(clientRedis, 1*time.Hour)

	postgresRepo := postgres.NewPostgresDestinationRepository(pool)
	repoApi := travelpayouts.NewTravelpayoutsRepository(token)
	cachedApiRepo := cache.NewCachedDestinationRepository(repoApi, cacheL1Repo, cacheRedisL2Repo, appLogger)

	//jwt
	postgresUserRepo := postgres.NewPostgresUserRepository(pool)
	authService := usecase.NewAuthService(postgresUserRepo, appLogger)
	authHandler := deliveryhttp.NewAuthHandler(authService)

	service := usecase.NewTravelService(appLogger, postgresRepo, cachedApiRepo)
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
		pgPool:     pool,
		log:        appLogger,
	}
}

func (a *App) Run() error {
	a.log.Info().Msg("start server port :8080")
	return a.httpServer.ListenAndServe()
}

func (a *App) Shutdown() {
	a.pgPool.Close()
	a.log.Info().Msg("Пул подключений к Postgres успешно закрыт")
}
