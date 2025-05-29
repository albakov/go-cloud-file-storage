package jwt

import (
	"crypto/rand"
	"encoding/base64"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Service struct {
	secret []byte
	conf   *Config
}

func NewService(conf *Config) *Service {
	return &Service{
		secret: []byte(conf.Secret),
		conf:   conf,
	}
}

func (j *Service) GenerateAccessToken(userId int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: strconv.Itoa(int(userId)),
		ExpiresAt: &jwt.NumericDate{
			Time: time.Now().Add(time.Minute * time.Duration(j.conf.ExpiresMinutes)),
		},
		IssuedAt: &jwt.NumericDate{
			Time: time.Now(),
		},
	})

	return token.SignedString(j.secret)
}

func (j *Service) ValidateAccessToken(tokenStr string) (*jwt.Token, error) {
	return jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return j.secret, nil
	})
}

func (j *Service) GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(b), nil
}
