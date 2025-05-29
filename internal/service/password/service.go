package password

import (
	"github.com/albakov/go-cloud-file-storage/internal/logger"
	"golang.org/x/crypto/bcrypt"
)

func CreateHashedPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", logger.Error("password", "CreateHashedPassword", err)
	}

	return string(hashed), nil
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return false
	}

	return true
}
