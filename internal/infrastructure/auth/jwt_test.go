package auth

import (
	"os"
	"testing"
)

func TestJWT_GenerateToken_Success(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret_key_123")
	defer os.Unsetenv("JWT_SECRET")

	tokenStr, err := GenerateToken(1, "test@mail.ru")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("Token is empty")
	}
}
