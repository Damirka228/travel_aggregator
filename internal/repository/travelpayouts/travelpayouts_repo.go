package travelpayouts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

func (r TravelpayoutsRepository) GetAll(ctx context.Context, search domain.FlightSearch) ([]domain.Destination, error) {
	departureDate := search.DepartureDate.Format("2006-01-02")
	returnDate := search.ReturnDate.Format("2006-01-02")

	params := url.Values{}
	params.Set("origin", search.Origin)
	params.Set("departure_at", departureDate)
	params.Set("return_at", returnDate)
	params.Set("one_way", "false")
	params.Set("currency", "rub")
	params.Set("sorting", "price")
	params.Set("unique", "false")
	params.Set("limit", "100")
	params.Set("page", "1")

	endpoint := "https://api.travelpayouts.com/aviasales/v3/prices_for_dates"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("create flight request: %w", err)
	}

	req.Header.Set("X-Access-Token", r.Token)

	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send flight request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"flight API returned status %d",
			resp.StatusCode,
		)
	}

	var parsed domain.PricesForDatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode flight response: %w", err)
	}

	if !parsed.Success {
		return nil, fmt.Errorf("flight API reported an unsuccessful request")
	}

	result := make([]domain.Destination, 0, len(parsed.Data))

	for _, offer := range parsed.Data {
		departureAt, err := time.Parse(time.RFC3339, offer.DepartureAt)
		if err != nil {
			continue
		}

		returnAt, err := time.Parse(time.RFC3339, offer.ReturnAt)
		if err != nil {
			continue
		}

		if offer.Origin != search.Origin || offer.Destination == "" {
			continue
		}

		if departureAt.Format("2006-01-02") != departureDate ||
			returnAt.Format("2006-01-02") != returnDate {
			continue
		}

		if offer.Price <= 0 {
			continue
		}

		// Считаем разницу календарных дат без времени и смещения зоны.
		departureDay := time.Date(
			departureAt.Year(), departureAt.Month(), departureAt.Day(),
			0, 0, 0, 0, time.UTC,
		)
		returnDay := time.Date(
			returnAt.Year(), returnAt.Month(), returnAt.Day(),
			0, 0, 0, 0, time.UTC,
		)

		days := int(returnDay.Sub(departureDay) / (24 * time.Hour))
		if days <= 0 {
			continue
		}

		result = append(result, domain.Destination{
			ID:          len(result) + 1,
			Origin:      offer.Origin,
			City:        offer.Destination,
			Price:       offer.Price,
			Days:        days,
			DepartureAt: departureAt,
			ReturnAt:    returnAt,
		})
	}

	return result, nil
}
