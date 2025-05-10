package controller

import (
	"fmt"
	"strconv"

	"github.com/albakov/go-cloud-file-storage/pkg/logger"
	"github.com/gofiber/fiber/v2"
)

func RequestedUserId(c *fiber.Ctx) int64 {
	const (
		f  = "controller"
		op = "RequestedUserId"
	)

	userId, err := strconv.ParseInt(c.Locals("user_id").(string), 10, 64)
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

func SetCommonHeaders(c *fiber.Ctx) {
	c.Accepts("application/json")
	c.Set(fiber.HeaderContentType, "application/json")
	c.Set(fiber.HeaderAccept, "application/json")
}
