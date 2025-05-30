package profile

import (
	"errors"
	"github.com/albakov/go-cloud-file-storage/internal/api/controller"
	"github.com/albakov/go-cloud-file-storage/internal/api/entity"
	"github.com/albakov/go-cloud-file-storage/internal/api/entity/profile"
	"github.com/albakov/go-cloud-file-storage/internal/logger"
	"github.com/albakov/go-cloud-file-storage/internal/storage"
	"github.com/albakov/go-cloud-file-storage/internal/storage/user"
	"github.com/gofiber/fiber/v2"
)

type Profile struct {
	pkg         string
	userService UserService
}

type UserService interface {
	UserById(userId int64) (user.User, error)
}

func New(userService UserService) *Profile {
	return &Profile{
		pkg:         "profile",
		userService: userService,
	}
}

// ShowHandler godoc
//
//	@Summary		Profile
//	@Description	Show profile info (email)
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string					true	"Authorization Bearer <ACCESS_TOKEN>"
//	@Success		200				{object}	profile.ProfileResponse	"Email address"
//	@Failure		401				{object}	entity.ErrorResponse	"Unauthorized"
//	@Router			/user/me [get]
func (p *Profile) ShowHandler(ctx *fiber.Ctx) error {
	const op = "ShowHandler"

	controller.SetCommonHeaders(ctx)

	userId := controller.RequestedUserId(ctx)
	us, err := p.userService.UserById(userId)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			logger.Add(p.pkg, op, err)
		}

		return ctx.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	ctx.Status(fiber.StatusOK)

	return ctx.JSON(&profile.ProfileResponse{
		Email: us.Email.String,
	})
}
