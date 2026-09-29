package usecase

import (
	"context"
	"errors"
	"log"

	"github.com/Damirka228/travel_aggregator/internal/domain"
	"github.com/Damirka228/travel_aggregator/internal/infrastructure/auth"
)

type AuthService struct {
	userRepo domain.UserRepository
}

func NewAuthService(repo domain.UserRepository) *AuthService {
	return &AuthService{
		userRepo: repo,
	}
}

func (a *AuthService) SignUp(ctx context.Context, email string, pass string) (string, error) {
	user, err := a.userRepo.GetUserByEmail(ctx, email)
	if err == nil && user.ID != 0 {
		return "", errors.New("user with this email already exists")
	}
	hashPass, err := auth.HashPassword(pass)
	if err != nil {
		log.Printf("error hashing password, err: %s", err)
		return "", err
	}
	user, err = a.userRepo.CreateUser(ctx, email, hashPass)
	if err != nil {
		log.Printf("error create user to DB, err: %s", err)
		return "", err
	}
	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		log.Printf("error to create hash-token, err: %s", err)
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
		log.Printf("error to create hash-token, err: %s", err)
		return "", err
	}
	return token, nil
}
