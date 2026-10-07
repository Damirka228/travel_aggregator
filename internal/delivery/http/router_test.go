package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"github.com/Damirka228/travel_aggregator/internal/usecase"
	"github.com/rs/zerolog"
)

func testRouter(enableProfiler bool) http.Handler {
	log := logger.Logger{Logger: zerolog.Nop()}
	travel := usecase.NewTravelService(log, &mockHotelRepo{}, &mockFlightRepo{})
	auth := usecase.NewAuthService(&mockUserRepository{}, log)
	return NewRouter(NewHandler(travel), NewAuthHandler(auth), enableProfiler).SetupRoutes()
}

func TestRouterRegistersAPIEndpoints(t *testing.T) {
	router := testRouter(false)
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/destinations?budget=100000&days=5&city=MOW&departure_date=2026-11-10", http.StatusOK},
		{http.MethodPost, "/auth/signup", http.StatusBadRequest},
		{http.MethodPost, "/auth/signin", http.StatusBadRequest},
		{http.MethodGet, "/auth/signup", http.StatusMethodNotAllowed},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString("{invalid"))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if response.Code != tc.status {
				t.Fatalf("expected status %d, got %d", tc.status, response.Code)
			}
			if tc.status == http.StatusOK {
				var result usecase.TravelResponse
				if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
					t.Fatal(err)
				}
				if len(result.BestTour) == 0 {
					t.Fatal("registered handler did not return the expected tour")
				}
			}
		})
	}
}

func TestRouterProfilerRequiresOptIn(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		response := httptest.NewRecorder()
		testRouter(enabled).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
		want := http.StatusNotFound
		if enabled {
			want = http.StatusOK
		}
		if response.Code != want {
			t.Fatalf("profiler enabled=%v: expected %d, got %d", enabled, want, response.Code)
		}
	}
}
