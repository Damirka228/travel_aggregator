package usecase

import (
	"context"
	"errors"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/auth"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/logger"
)

type AuthService struct {
	userRepo domain.UserRepository
	log      logger.Logger
}

func NewAuthService(repo domain.UserRepository, log logger.Logger) *AuthService {
	return &AuthService{
		userRepo: repo,
		log:      log,
	}
}

func (a *AuthService) SignUp(ctx context.Context, email string, pass string) (string, error) {
	user, err := a.userRepo.GetUserByEmail(ctx, email)
	if err == nil && user.ID != 0 {
		return "", errors.New("user with this email already exists")
	}
	hashPass, err := auth.HashPassword(pass)
	if err != nil {
		a.log.Error().Err(err).Str("email", email).Msg("Ошибка при хэшировании пароля")
		return "", err
	}
	user, err = a.userRepo.CreateUser(ctx, email, hashPass)
	if err != nil {
		a.log.Error().Err(err).Str("email", email).Msg("Не удалось сохранить пользователя в Postgres")
		return "", err
	}
	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		a.log.Error().Err(err).Str("email", email).Msg("Ошибка генерации JWT токена при регистрации")
		return "", err
	}
	return token, nil
}

func (a *AuthService) SignIn(ctx context.Context, email string, pass string) (string, error) {
	user, err := a.userRepo.GetUserByEmail(ctx, email)
	if user.ID == 0 || err != nil {
		return "", errors.New("invalid email or password")
	}

	ok := auth.CheckHashPassword(pass, user.PasswordHash)
	if !ok {
		return "", errors.New("invalid email or password")
	}

	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		a.log.Error().Err(err).Str("email", email).Msg("Ошибка генерации JWT токена при входе")
		return "", err
	}
	return token, nil
}
