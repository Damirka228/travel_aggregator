package travelpayouts

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Damirka228/travel_aggregator/internal/domain"
)

type flightTestTransport func(*http.Request) (*http.Response, error)

func (f flightTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func flightTestResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func flightTestSearch() domain.FlightSearch {
	return domain.FlightSearch{Origin: "MOW", DepartureDate: time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC), ReturnDate: time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)}
}

func TestTravelpayouts_RequestParametersAndOfferMapping(t *testing.T) {
	search := flightTestSearch()
	valid := domain.FlightPrice{Origin: "MOW", Destination: "KZN", Price: 5000, DepartureAt: "2026-11-10T23:00:00+03:00", ReturnAt: "2026-11-17T08:00:00+03:00"}
	offers := []domain.FlightPrice{valid}
	second := valid
	second.Price = 6000
	offers = append(offers, second)
	for _, change := range []func(*domain.FlightPrice){
		func(o *domain.FlightPrice) { o.DepartureAt = "bad" },
		func(o *domain.FlightPrice) { o.ReturnAt = "" },
		func(o *domain.FlightPrice) { o.Origin = "LED" },
		func(o *domain.FlightPrice) { o.Destination = "" },
		func(o *domain.FlightPrice) { o.DepartureAt = "2026-11-11T09:00:00+03:00" },
		func(o *domain.FlightPrice) { o.ReturnAt = "2026-11-18T09:00:00+03:00" },
		func(o *domain.FlightPrice) { o.Price = 0 },
		func(o *domain.FlightPrice) { o.Price = -1 },
	} {
		invalid := valid
		change(&invalid)
		offers = append(offers, invalid)
	}
	body, err := json.Marshal(domain.PricesForDatesResponse{Success: true, Data: offers})
	if err != nil {
		t.Fatal(err)
	}
	called := false
	repo := NewTravelpayoutsRepository("test-token")
	repo.Client.Transport = flightTestTransport(func(req *http.Request) (*http.Response, error) {
		called = true
		if req.Method != http.MethodGet || req.URL.Scheme != "https" || req.URL.Host != "api.travelpayouts.com" || req.URL.Path != "/aviasales/v3/prices_for_dates" {
			t.Fatalf("unexpected endpoint: %s %s", req.Method, req.URL)
		}
		query := req.URL.Query()
		want := map[string]string{"origin": "MOW", "departure_at": "2026-11-10", "return_at": "2026-11-17", "one_way": "false", "currency": "rub", "sorting": "price", "unique": "false", "limit": "100", "page": "1"}
		if len(query) != len(want) {
			t.Fatalf("unexpected query parameters: %v", query)
		}
		for key, value := range want {
			if query.Get(key) != value {
				t.Fatalf("parameter %s: got %q, want %q", key, query.Get(key), value)
			}
		}
		if req.Header.Get("X-Access-Token") != "test-token" || query.Has("token") {
			t.Fatal("token must be passed through the header")
		}
		return flightTestResponse(http.StatusOK, string(body)), nil
	})
	got, err := repo.GetAll(context.Background(), search)
	if err != nil || !called || len(got) != 2 {
		t.Fatalf("unexpected offers: %+v err=%v called=%v", got, err, called)
	}
	for i, offer := range got {
		if offer.ID != i+1 || offer.Origin != "MOW" || offer.City != "KZN" || offer.Price != float64(5000+i*1000) || offer.Days != 7 || offer.Country != "" {
			t.Fatalf("incorrect offer mapping: %+v", offer)
		}
		if offer.DepartureAt.Format(time.RFC3339) != valid.DepartureAt || offer.ReturnAt.Format(time.RFC3339) != valid.ReturnAt {
			t.Fatalf("flight times or offsets were lost: %+v", offer)
		}
	}
}

func TestTravelpayouts_ResponseErrorsAndEmptyResult(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{"HTTP failure", http.StatusServiceUnavailable, `{}`, true},
		{"API failure", http.StatusOK, `{"success":false,"data":[]}`, true},
		{"malformed JSON", http.StatusOK, `{`, true},
		{"wrong data shape", http.StatusOK, `{"success":true,"data":{}}`, true},
		{"no cached offers", http.StatusOK, `{"success":true,"data":[]}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := NewTravelpayoutsRepository("test-token")
			repo.Client.Transport = flightTestTransport(func(*http.Request) (*http.Response, error) { return flightTestResponse(tc.status, tc.body), nil })
			got, err := repo.GetAll(context.Background(), flightTestSearch())
			if (err != nil) != tc.wantErr || len(got) != 0 {
				t.Fatalf("unexpected response: %+v err=%v", got, err)
			}
		})
	}
}

func TestTravelpayouts_PreservesTransportError(t *testing.T) {
	transportErr := errors.New("test transport unavailable")
	repo := NewTravelpayoutsRepository("test-token")
	repo.Client.Transport = flightTestTransport(func(*http.Request) (*http.Response, error) { return nil, transportErr })
	if _, err := repo.GetAll(context.Background(), flightTestSearch()); !errors.Is(err, transportErr) {
		t.Fatalf("transport error was lost: %v", err)
	}
}

func TestTravelpayouts_PropagatesContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repo := NewTravelpayoutsRepository("test-token")
	repo.Client.Transport = flightTestTransport(func(req *http.Request) (*http.Response, error) {
		if req.Context().Err() == nil {
			t.Error("request did not receive canceled context")
		}
		return nil, req.Context().Err()
	})
	if _, err := repo.GetAll(ctx, flightTestSearch()); !errors.Is(err, context.Canceled) {
		t.Fatalf("context cancellation was lost: %v", err)
	}
}
