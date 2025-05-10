package jwt

import (
	"crypto/rand"
	"encoding/base64"
	"strconv"
	"time"

	"github.com/albakov/go-cloud-file-storage/pkg/config"
	"github.com/golang-jwt/jwt/v5"
)

type JWT struct {
	secret []byte
	c      *config.Config
}

func MustNew(c *config.Config) *JWT {
	return &JWT{
		secret: []byte(c.JWTSecret),
		c:      c,
	}
}

func (j *JWT) GenerateAccessToken(userId int64) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: strconv.Itoa(int(userId)),
		ExpiresAt: &jwt.NumericDate{
			Time: time.Now().Add(time.Minute * time.Duration(j.c.JWTExpiresMinutes)),
		},
		IssuedAt: &jwt.NumericDate{
			Time: time.Now(),
		},
	})

	return token.SignedString(j.secret)
}

func (j *JWT) ValidateAccessToken(tokenStr string) (*jwt.Token, error) {
	return jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return j.secret, nil
	})
}

func (j *JWT) GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(b), nil
}
