package profile

import (
	"database/sql"
	"errors"
	"github.com/albakov/go-cloud-file-storage/pkg/api/entity"
	"github.com/albakov/go-cloud-file-storage/pkg/api/entity/profile_entity"

	"github.com/albakov/go-cloud-file-storage/pkg/api/controller"
	"github.com/albakov/go-cloud-file-storage/pkg/logger"
	"github.com/albakov/go-cloud-file-storage/pkg/storage"
	"github.com/albakov/go-cloud-file-storage/pkg/storage/entity/user"
	"github.com/albakov/go-cloud-file-storage/pkg/storage/mariadb/user_storage"
	"github.com/gofiber/fiber/v2"
)

type Profile struct {
	f           string
	userStorage UserStorage
}

type UserStorage interface {
	ById(id int64) (user.User, error)
}

func New(db *sql.DB) *Profile {
	return &Profile{
		f:           "profile",
		userStorage: user_storage.New(db),
	}
}

// ShowHandler godoc
//
//	@Summary		Profile
//	@Description	Show profile info (email)
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string							true	"Authorization Bearer <ACCESS_TOKEN>"
//	@Success		200				{object}	profile_entity.ProfileResponse	"Email address"
//	@Failure		401				{object}	entity.ErrorResponse			"Unauthorized"
//	@Router			/user/me [get]
func (p *Profile) ShowHandler(c *fiber.Ctx) error {
	const op = "ShowHandler"

	controller.SetCommonHeaders(c)

	userId := controller.RequestedUserId(c)
	if userId == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	us, err := p.userStorage.ById(userId)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			logger.Add(p.f, op, err)
		}

		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	c.Status(fiber.StatusOK)

	return c.JSON(&profile_entity.ProfileResponse{
		Email: us.Email.String,
	})
}
