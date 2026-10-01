package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/auth"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
	"github.com/Damirka228/travel_aggregator/internal/usecase"
)

type mockUserRepository struct {
	userExists bool
}

func (m *mockUserRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	if m.userExists && email == "exist@mail.ru" {
		hash, _ := auth.HashPassword("pass123")
		return domain.User{ID: 1, Email: email, PasswordHash: hash}, nil
	}
	return domain.User{}, errors.New("user not found")
}

func (m *mockUserRepository) CreateUser(ctx context.Context, email string, passwordHash string) (domain.User, error) {
	return domain.User{ID: 1, Email: email, PasswordHash: passwordHash}, nil
}

func TestAuthHandler_SignUp_Success(t *testing.T) {
	log := logger.New()
	service := usecase.NewAuthService(&mockUserRepository{userExists: false}, log)
	h := NewAuthHandler(service)

	body, _ := json.Marshal(map[string]string{"email": "new@mail.ru", "password": "password123"})
	req, err := http.NewRequest("POST", "/auth/signup", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	h.SignUp(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Ожидался статус 200 на регистрацию, получен: %v", rr.Code)
	}
}

func TestAuthHandler_SignUp_EmptyFields(t *testing.T) {
	log := logger.New()
	service := usecase.NewAuthService(&mockUserRepository{userExists: false}, log)
	h := NewAuthHandler(service)

	body, _ := json.Marshal(map[string]string{"email": "", "password": ""})
	req, _ := http.NewRequest("POST", "/auth/signup", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	h.SignUp(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Ожидался статус 400 при пустом email/пароле, получен: %v", rr.Code)
	}
}

func TestAuthHandler_SignIn_Success(t *testing.T) {
	log := logger.New()
	service := usecase.NewAuthService(&mockUserRepository{userExists: true}, log)
	h := NewAuthHandler(service)

	body, _ := json.Marshal(map[string]string{"email": "exist@mail.ru", "password": "pass123"})
	req, _ := http.NewRequest("POST", "/auth/signin", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	h.SignIn(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Ожидался статус 200 на логин, получен: %v", rr.Code)
	}
}

func TestAuthHandler_SignIn_InvalidJSON(t *testing.T) {
	log := logger.New()
	service := usecase.NewAuthService(&mockUserRepository{userExists: true}, log)
	h := NewAuthHandler(service)

	req, _ := http.NewRequest("POST", "/auth/signin", bytes.NewBufferString("{broken-json: true"))
	rr := httptest.NewRecorder()

	h.SignIn(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Ожидался статус 400 при битом теле JSON, получен: %v", rr.Code)
	}
}
