package middleware

import (
	"github.com/albakov/go-cloud-file-storage/internal/api/controller"
	"github.com/albakov/go-cloud-file-storage/internal/api/entity"
	"github.com/albakov/go-cloud-file-storage/internal/service/jwt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type Middleware struct {
	jwt *jwt.JWT
}

func New(j *jwt.JWT) *Middleware {
	return &Middleware{
		jwt: j,
	}
}

func (m *Middleware) AuthenticatedMiddleware(c *fiber.Ctx) error {
	t, err := m.jwt.ValidateAccessToken(strings.TrimPrefix(c.Get("Authorization"), "Bearer "))
	if err != nil || !t.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	userId, err := t.Claims.GetSubject()
	if err != nil || userId == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	c.Locals("user_id", userId)

	return c.Next()
}
