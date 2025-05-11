package password

import (
	"github.com/albakov/go-cloud-file-storage/internal/logger"
	"golang.org/x/crypto/bcrypt"
)

const f = "password"

func CreateHashedPassword(password string) (string, error) {
	const op = "CreateHashedPassword"

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", logger.Error(f, op, err)
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
