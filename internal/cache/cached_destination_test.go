package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
)

type mockApiRepo struct {
	destination []domain.Destination
	err         error
}

func (m *mockApiRepo) GetAll(ctx context.Context, origin string) ([]domain.Destination, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.destination, nil
}

type mockL2Cache struct {
	store map[string][]domain.Destination
}

func (m *mockL2Cache) Get(ctx context.Context, key string) ([]domain.Destination, bool) {
	data, ok := m.store[key]
	return data, ok
}
func (m *mockL2Cache) Set(ctx context.Context, key string, val []domain.Destination) error {
	m.store[key] = val
	return nil
}

// TestCache_AllMiss_FetchFromAPI — Полный промах кэшей. Сервер идет в API и кэширует всё!
func TestCache_AllMiss_FetchFromAPI(t *testing.T) {
	log := logger.New()
	apiData := []domain.Destination{{ID: 1, City: "KZN", Price: 3000}}

	apiMock := &mockApiRepo{destination: apiData}
	l2Mock := &mockL2Cache{store: make(map[string][]domain.Destination)}
	l1Cache := NewSharedCache(2, 1*time.Minute)

	cachedRepo := NewCachedDestinationRepository(apiMock, l1Cache, l2Mock, log)

	res, err := cachedRepo.GetAll(context.Background(), "MOW")
	if err != nil {
		t.Fatalf("Ожидался успех, получено: %v", err)
	}

	if len(res) != 1 || res[0].City != "KZN" {
		t.Error("Данные из API не долетели до юзера")
	}

	if _, ok := l2Mock.store["destinations:MOW"]; !ok {
		t.Error("Данные не закешировались в L2 (Redis) при промахе")
	}
}

// TestCache_L1Hit — Попадание в L1. Сервер берет данные мгновенно из ОЗУ!
func TestCache_L1Hit(t *testing.T) {
	log := logger.New()
	apiMock := &mockApiRepo{err: errors.New("интернет упал!")}
	l2Mock := &mockL2Cache{store: make(map[string][]domain.Destination)}
	l1Cache := NewSharedCache(2, 1*time.Minute)

	mockData := []domain.Destination{{ID: 2, City: "LED", Price: 5000}}
	l1Cache.Set("destinations:MOW", mockData)

	cachedRepo := NewCachedDestinationRepository(apiMock, l1Cache, l2Mock, log)

	res, err := cachedRepo.GetAll(context.Background(), "MOW")
	if err != nil {
		t.Fatalf("Ожидался успех из L1, получено: %v", err)
	}

	if len(res) != 1 || res[0].City != "LED" {
		t.Error("Данные из L1 кэша не вернулись")
	}
}
