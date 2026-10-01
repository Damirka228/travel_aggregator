package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
)

type testFlightRepository struct {
	flights []domain.Destination
	err     error
}

func (t *testFlightRepository) GetAll(ctx context.Context, origin string) ([]domain.Destination, error) {
	if t.err != nil {
		return nil, t.err
	}
	return t.flights, nil
}

// 1. Тест на идеальный сценарий (Топ-3 билета)
func TestFindDestinations_Success(t *testing.T) {
	log := logger.New()
	mockData := []domain.Destination{
		{ID: 1, City: "KZN", Country: "International", Price: 10000, Days: 5},
		{ID: 2, City: "LED", Country: "International", Price: 30000, Days: 7},
		{ID: 3, City: "KUF", Country: "International", Price: 15000, Days: 3},
	}
	mockRepo := &testFlightRepository{flights: mockData}
	service := NewTravelService(log, mockRepo)

	res, err := service.FindDestinations(context.Background(), 40000, 7, "MOW")
	if err != nil {
		t.Fatalf("Ожидался успех, получено: %v", err)
	}
	if res.BestOption.City == "" || len(res.Cheapest) == 0 || len(res.Longest) == 0 {
		t.Error("Результаты не должны быть пустыми")
	}
}

func TestFindDestinations_ArraySlicing(t *testing.T) {
	log := logger.New()

	// Генерируем сразу 10 билетов, чтобы массивы переполнились!
	var mockData []domain.Destination
	for i := 1; i <= 10; i++ {
		mockData = append(mockData, domain.Destination{
			ID: i, City: "KZN", Country: "International", Price: float64(1000 * i), Days: i,
		})
	}

	mockRepo := &testFlightRepository{flights: mockData}
	service := NewTravelService(log, mockRepo)

	res, err := service.FindDestinations(context.Background(), 100000, 30, "MOW")
	if err != nil {
		t.Fatalf("Ожидался успех, получено: %v", err)
	}
	if len(res.Cheapest) != 5 {
		t.Errorf("Ожидалось ровно 5 дешевых билетов, получено: %d", len(res.Cheapest))
	}
	if len(res.Longest) != 3 {
		t.Errorf("Ожидалось ровно 3 долгих билета, получено: %d", len(res.Longest))
	}
}

func TestFindDestinations_RepoError(t *testing.T) {
	log := logger.New()

	mockRepo := &testFlightRepository{err: errors.New("database connection refused")}
	service := NewTravelService(log, mockRepo)

	res, err := service.FindDestinations(context.Background(), 50000, 7, "MOW")

	if err != nil {
		t.Fatalf("Сервер не должен падать с ошибкой при отказе одного источника, получено: %v", err)
	}

	if res.BestOption.ID != 0 {
		t.Error("Ожидался пустой ответ при ошибке репозитория")
	}
}

func TestFindDestinations_Empty(t *testing.T) {
	log := logger.New()
	mockData := []domain.Destination{{ID: 1, City: "PAR", Price: 100000, Days: 5}}
	mockRepo := &testFlightRepository{flights: mockData}
	service := NewTravelService(log, mockRepo)

	res, err := service.FindDestinations(context.Background(), 20000, 7, "MOW")
	if err != nil {
		t.Fatalf("Ожидался пустой ответ, получено: %v", err)
	}
	if res.BestOption.ID != 0 {
		t.Error("Результат должен быть пустым")
	}
}
