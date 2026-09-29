package domain

import "context"


type User struct {
	ID int
	Email string
	PasswordHash string
}


type UserRepository interface {
	CreateUser(ctx context.Context, email string, passwordHash string) (User,error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
}