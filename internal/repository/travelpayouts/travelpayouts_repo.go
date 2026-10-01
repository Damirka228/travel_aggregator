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
	idCounter := 1

	for iataCode, options := range parsed.Data {
		for _, opt := range options {
			realDays := 7

			if opt.DepartureAt != "" && opt.ReturnAt != "" {
				depTime, err1 := time.Parse("2006-01-02", opt.DepartureAt[:10])
				retTime, err2 := time.Parse("2006-01-02", opt.ReturnAt[:10])

				if err1 == nil && err2 == nil {
					hours := retTime.Sub(depTime).Hours()
					calculatedDays := int(hours / 24)
					if calculatedDays > 0 {
						realDays = calculatedDays
					}
				}
			}

			result = append(result, domain.Destination{
				ID:      idCounter,
				City:    iataCode,
				Country: "International",
				Price:   opt.Price,
				Days:    realDays,
			})

			idCounter++
			break
		}
	}

	log.Printf("API нашло билетов: %d штук\n", len(result))
	return result, nil
}
