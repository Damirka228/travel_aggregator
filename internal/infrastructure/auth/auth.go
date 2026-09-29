package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(pass string) (string, error) {
	bytesPass, err := bcrypt.GenerateFromPassword([]byte(pass), 10)

	if err != nil {
		return "", err
	}

	return string(bytesPass), nil
}

func CheckHashPassword (pass string, hashPass string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashPass),[]byte(pass))
	return err == nil
}