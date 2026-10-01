package auth

import "testing"

func TestAuth_HashAndCheck_Success(t *testing.T) {
	password := "my_secret_pass_123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Не удалось сгенерировать хэш пароля: %v", err)
	}
	if hash == "" {
		t.Fatal("Хэш прилетел пустым")
	}

	isValid := CheckHashPassword(password, hash)
	if !isValid {
		t.Error("Валидация хэша провалена для правильного пароля")
	}
}

func TestAuth_Check_WrongPassword(t *testing.T) {
	correctPassword := "correct_password"
	wrongPassword := "hacker_attack_123"

	hash, err := HashPassword(correctPassword)
	if err != nil {
		t.Fatalf("Не удалось сгенерировать хэш: %v", err)
	}

	// 2. Проверяем хэш с НЕВЕРНЫМ паролем
	isValid := CheckHashPassword(wrongPassword, hash)
	if isValid {
		t.Error("Система безопасности пробита! Неверный пароль был принят за валидный")
	}
}
