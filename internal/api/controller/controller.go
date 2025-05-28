package controller

import (
	"fmt"
	"github.com/albakov/go-cloud-file-storage/internal/logger"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func RequestedUserId(ctx *fiber.Ctx) int64 {
	const (
		f  = "controller"
		op = "RequestedUserId"
	)

	userId, err := strconv.ParseInt(ctx.Locals("user_id").(string), 10, 64)
	if err != nil {
		logger.Add(f, op, err)

		return 0
	}

	if userId == 0 {
		logger.Add(f, op, fmt.Errorf("invalid user id"))

		return 0
	}

	return userId
}

func SetCommonHeaders(ctx *fiber.Ctx) {
	ctx.Accepts("application/json")
	ctx.Set(fiber.HeaderContentType, "application/json")
	ctx.Set(fiber.HeaderAccept, "application/json")
}
