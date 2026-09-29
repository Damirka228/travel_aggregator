package http

import (
	"encoding/json"
	"net/http"

	"github.com/Damirka228/travel_aggregator/internal/usecase"
)

type AuthHandler struct {
	authHandler *usecase.AuthService
}

func NewAuthHandler(service *usecase.AuthService) *AuthHandler {
	return &AuthHandler{
		authHandler: service,
	}
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email or password empty", http.StatusBadRequest)
		return
	}
	token, err := h.authHandler.SignUp(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (h *AuthHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		http.Error(w, "email or password empty", http.StatusBadRequest)
		return
	}
	token, err := h.authHandler.SignIn(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
