package auth

import (
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

func GenerateToken(userID int, email string) (string, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден")
	}

	jwtSecret := os.Getenv("JWT_SECRET")

	claims := jwt.MapClaims{
		"user-id": userID,
		"email":   email,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}
