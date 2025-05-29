package validation

import (
	"github.com/albakov/go-cloud-file-storage/internal/api/controller"
	"github.com/albakov/go-cloud-file-storage/internal/api/entity"
	"github.com/albakov/go-cloud-file-storage/internal/api/entity/profile"
	"github.com/gofiber/fiber/v2"
)

func EmailAndPasswordValidation(ctx *fiber.Ctx) error {
	var r profile.LoginRequest
	err := ctx.BodyParser(&r)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	if !isEmailAndPasswordValid(r.Email, r.Password) {
		return ctx.Status(fiber.StatusUnauthorized).JSON(
			&entity.ErrorResponse{Message: controller.MessageLoginOrPasswordInvalid},
		)
	}

	ctx.Locals("user_data", r)

	return ctx.Next()
}

func isEmailAndPasswordValid(email, password string) bool {
	return email != "" && password != ""
}
