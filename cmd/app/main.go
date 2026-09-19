package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Damirka228/travel_aggregator/internal/domain"
)

func main() {
	http.HandleFunc("/destinations", destinationsHandler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("error start server:", err)
	}

}

func Convert(value string) (int, error) {
	val, err := strconv.Atoi(value)
	if err != nil {
		fmt.Println("error convertation budget: ", err)
		return 0, err
	}
	return val, nil
}

func destinationsHandler(w http.ResponseWriter, r *http.Request) {
	budget := r.URL.Query().Get("budget")
	budgetStr, err := Convert(budget)
	if err != nil{
		http.Error(w, "invalid budget", http.StatusBadRequest)
		return
	}
	days := r.URL.Query().Get("days")
	daysStr, err := Convert(days)
	if err != nil{
		http.Error(w, "invalid budget", http.StatusBadRequest)
		return
	}

	result := domain.FindDestinations(float64(budgetStr), daysStr)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
