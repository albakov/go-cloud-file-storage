package controller

import (
	"github.com/albakov/go-cloud-file-storage/internal/api/entity/profile"
	"github.com/gofiber/fiber/v2"
)

func RequestedUserId(ctx *fiber.Ctx) int64 {
	return ctx.Locals("user_id").(int64)
}

func RequestedLogin(ctx *fiber.Ctx) profile.LoginRequest {
	return ctx.Locals("user_data").(profile.LoginRequest)
}

func SetCommonHeaders(ctx *fiber.Ctx) {
	ctx.Accepts("application/json")
	ctx.Set(fiber.HeaderContentType, "application/json")
	ctx.Set(fiber.HeaderAccept, "application/json")
}
