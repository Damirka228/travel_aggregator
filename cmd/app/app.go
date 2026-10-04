package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/cache"
	deliveryhttp "github.com/Damirka228/travel_aggregator/internal/delivery/http"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"github.com/Damirka228/travel_aggregator/internal/repository/hotels"
	"github.com/Damirka228/travel_aggregator/internal/repository/postgres"
	"github.com/Damirka228/travel_aggregator/internal/repository/redis"
	"github.com/Damirka228/travel_aggregator/internal/repository/travelpayouts"
	"github.com/Damirka228/travel_aggregator/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	goredis "github.com/redis/go-redis/v9"
)

const (
	startupTimeout  = 10 * time.Second
	shutdownTimeout = 15 * time.Second
)

type App struct {
	httpServer   *http.Server
	pgPool       *pgxpool.Pool
	redisClient  *goredis.Client
	log          logger.Logger
	shutdownOnce sync.Once
	shutdownErr  error
}

func NewApp(ctx context.Context) (_ *App, err error) {
	app := &App{log: logger.New()}
	defer func() {
		if err != nil {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
			defer cancel()
			err = errors.Join(err, app.Shutdown(cleanupCtx))
		}
	}()

	if loadErr := godotenv.Load(); loadErr != nil {
		if !os.IsNotExist(loadErr) {
			return nil, fmt.Errorf("load .env: %w", loadErr)
		}
		app.log.Warn().Msg("Файл .env не найден; используются переменные окружения")
	}

	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	startupCtx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()

	app.pgPool, err = pgxpool.New(startupCtx, cfg.databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create Postgres pool: %w", err)
	}
	if err := app.pgPool.Ping(startupCtx); err != nil {
		return nil, fmt.Errorf("connect to Postgres: %w", err)
	}

	app.redisClient, err = redis.NewClient(startupCtx, cfg.redisAddr)
	if err != nil {
		return nil, fmt.Errorf("connect to Redis: %w", err)
	}

	cacheL1Repo := cache.NewSharedCache(256, 5*time.Minute)
	cacheRedisL2Repo := redis.NewDestinationsCache(app.redisClient, time.Hour)
	hotelRepo := hotels.NewHotelRepository()
	postgresRepo := postgres.NewPostgresDestinationRepository(app.pgPool)
	repoAPI := travelpayouts.NewTravelpayoutsRepository(cfg.travelpayoutsToken)
	cachedAPIRepo := cache.NewCachedDestinationRepository(repoAPI, cacheL1Repo, cacheRedisL2Repo, app.log)

	postgresUserRepo := postgres.NewPostgresUserRepository(app.pgPool)
	authService := usecase.NewAuthService(postgresUserRepo, app.log)
	authHandler := deliveryhttp.NewAuthHandler(authService)

	service := usecase.NewTravelService(app.log, hotelRepo, postgresRepo, cachedAPIRepo)
	handler := deliveryhttp.NewHandler(service)
	router := deliveryhttp.NewRouter(handler, authHandler, cfg.enableProfiler)

	app.httpServer = &http.Server{
		Addr:              ":" + cfg.port,
		Handler:           router.SetupRoutes(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       time.Minute,
	}

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	a.log.Info().Str("addr", a.httpServer.Addr).Msg("HTTP-сервер запускается")

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- a.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		a.log.Info().Msg("Получен запрос на остановку приложения")
		return nil
	case err := <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	}
}

// Shutdown drains HTTP requests before closing the resources used by handlers.
// It is safe to call more than once, including after partial initialization.
func (a *App) Shutdown(ctx context.Context) error {
	a.shutdownOnce.Do(func() {
		if a.httpServer != nil {
			a.log.Info().Msg("Ожидание завершения текущих HTTP-запросов")
			if err := a.httpServer.Shutdown(ctx); err != nil {
				a.log.Error().Err(err).Msg("Время graceful shutdown истекло; соединения закрываются принудительно")
				a.shutdownErr = errors.Join(a.shutdownErr, fmt.Errorf("shutdown HTTP: %w", err))
				if closeErr := a.httpServer.Close(); closeErr != nil {
					a.shutdownErr = errors.Join(a.shutdownErr, fmt.Errorf("close HTTP: %w", closeErr))
				}
			}
		}

		if a.pgPool != nil {
			a.pgPool.Close()
		}
		if a.redisClient != nil {
			if err := a.redisClient.Close(); err != nil {
				a.shutdownErr = errors.Join(a.shutdownErr, fmt.Errorf("close Redis: %w", err))
			}
		}

		a.log.Info().Msg("Ресурсы приложения закрыты")
		if err := a.log.Close(); err != nil {
			a.shutdownErr = errors.Join(a.shutdownErr, fmt.Errorf("close logger: %w", err))
		}
	})

	return a.shutdownErr
}
