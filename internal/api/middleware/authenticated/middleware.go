package authenticated

import (
	"github.com/albakov/go-cloud-file-storage/internal/api/controller"
	"github.com/albakov/go-cloud-file-storage/internal/api/entity"
	"github.com/golang-jwt/jwt/v5"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type AuthService interface {
	ValidateAccessToken(tokenStr string) (*jwt.Token, error)
}

type Authenticated struct {
	authService AuthService
}

func New(authService AuthService) *Authenticated {
	return &Authenticated{
		authService: authService,
	}
}

func (a *Authenticated) Authenticated(ctx *fiber.Ctx) error {
	t, err := a.authService.ValidateAccessToken(strings.TrimPrefix(ctx.Get("Authorization"), "Bearer "))
	if err != nil || !t.Valid {
		return ctx.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	id, err := t.Claims.GetSubject()
	if err != nil || id == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	userId, err := strconv.ParseInt(id, 10, 64)
	if err != nil || userId == 0 {
		return ctx.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	ctx.Locals("user_id", userId)

	return ctx.Next()
}
