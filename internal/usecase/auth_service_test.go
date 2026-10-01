package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/auth"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
)

type mockUserRepository struct {
	users map[string]domain.User
}

func (m *mockUserRepository) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	user, ok := m.users[email]
	if !ok {
		return domain.User{}, errors.New("user not found")
	}
	return user, nil
}

func (m *mockUserRepository) CreateUser(ctx context.Context, email string, passwordHash string) (domain.User, error) {
	if _, ok := m.users[email]; ok {
		return domain.User{}, errors.New("user already exists")
	}
	newUser := domain.User{
		ID:           1,
		Email:        email,
		PasswordHash: passwordHash,
	}
	m.users[email] = newUser
	return newUser, nil
}

func TestAuthService_SignUp_Success(t *testing.T) {
	log := logger.New()
	mockRepo := &mockUserRepository{users: make(map[string]domain.User)}
	service := NewAuthService(mockRepo, log)

	token, err := service.SignUp(context.Background(), "test@mail.ru", "qwerty12345")

	if err != nil {
		t.Fatalf("Ожидалась успешная регистрация, получена ошибка: %v", err)
	}
	if token == "" {
		t.Error("Сервер должен был вернуть сгенерированный JWT токен, прилетела пустота")
	}
}

func TestAuthService_SignUp_UserExists(t *testing.T) {
	log := logger.New()
	mockRepo := &mockUserRepository{users: make(map[string]domain.User)}

	hash, _ := auth.HashPassword("password123")
	mockRepo.users["existing@mail.ru"] = domain.User{ID: 1, Email: "existing@mail.ru", PasswordHash: hash}

	service := NewAuthService(mockRepo, log)

	_, err := service.SignUp(context.Background(), "existing@mail.ru", "newpassword")

	if err == nil {
		t.Error("Ожидалась ошибка 'user with this email already exists', но запрос прошел успешно")
	}
}

func TestAuthService_SignIn_Success(t *testing.T) {
	log := logger.New()
	mockRepo := &mockUserRepository{users: make(map[string]domain.User)}

	rawPassword := "correct_pass"
	hash, _ := auth.HashPassword(rawPassword)
	mockRepo.users["user@mail.ru"] = domain.User{ID: 1, Email: "user@mail.ru", PasswordHash: hash}

	service := NewAuthService(mockRepo, log)

	token, err := service.SignIn(context.Background(), "user@mail.ru", rawPassword)

	if err != nil {
		t.Fatalf("Ожидался успешный вход, получена ошибка: %v", err)
	}
	if token == "" {
		t.Error("Токен при успешном входе не должен быть пустым")
	}
}

func TestAuthService_SignIn_WrongPassword(t *testing.T) {
	log := logger.New()
	mockRepo := &mockUserRepository{users: make(map[string]domain.User)}

	hash, _ := auth.HashPassword("valid_password")
	mockRepo.users["user@mail.ru"] = domain.User{ID: 1, Email: "user@mail.ru", PasswordHash: hash}

	service := NewAuthService(mockRepo, log)

	_, err := service.SignIn(context.Background(), "user@mail.ru", "hacker_attack")

	if err == nil {
		t.Error("Ожидалась ошибка 'invalid email or password', но сервер пустил пользователя")
	}
}
