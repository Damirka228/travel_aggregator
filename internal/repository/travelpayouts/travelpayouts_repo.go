package travelpayouts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/domain"
)

type TravelpayoutsRepository struct {
	Token  string
	Client *http.Client
}

func NewTravelpayoutsRepository(token string) *TravelpayoutsRepository {
	return &TravelpayoutsRepository{
		Token:  token,
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (r TravelpayoutsRepository) GetAll(ctx context.Context, origin string) ([]domain.Destination, error) {
	url := fmt.Sprintf("https://api.travelpayouts.com/v1/prices/cheap?origin=%s&destination=-&token=%s", origin, r.Token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании запроса: %w", err)
	}
	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}
	var parsed domain.CheapPriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}
	var result []domain.Destination
	for iataCode, options := range parsed.Data {
		for _, opt := range options {
			result = append(result, domain.Destination{
				City:    iataCode,
				Country: "International",
				Price:   opt.Price,
				Days:    7,
			})
			break
		}
	}
	log.Printf("API нашло билетов: %d штук\n", len(result))
	return result, nil
}
