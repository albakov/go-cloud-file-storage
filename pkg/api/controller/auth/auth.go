package auth

import (
	"database/sql"
	"errors"
	"github.com/albakov/go-cloud-file-storage/pkg/api/entity"
	"github.com/albakov/go-cloud-file-storage/pkg/service/jwt"
	"github.com/albakov/go-cloud-file-storage/pkg/service/password"
	"github.com/albakov/go-cloud-file-storage/pkg/service/user_service"
	"github.com/albakov/go-cloud-file-storage/pkg/service/user_service/user_entity"
	"github.com/albakov/go-cloud-file-storage/pkg/service/user_session_service"
	"github.com/albakov/go-cloud-file-storage/pkg/service/user_session_service/user_session_entity"
	"time"

	"github.com/albakov/go-cloud-file-storage/pkg/api/controller"
	"github.com/albakov/go-cloud-file-storage/pkg/api/entity/profile_entity"
	"github.com/albakov/go-cloud-file-storage/pkg/config"
	"github.com/albakov/go-cloud-file-storage/pkg/logger"
	"github.com/albakov/go-cloud-file-storage/pkg/storage"
	"github.com/albakov/go-cloud-file-storage/pkg/storage/entity/user"
	"github.com/gofiber/fiber/v2"
)

type Auth struct {
	f                  string
	c                  *config.Config
	jwt                *jwt.JWT
	userService        UserService
	userSessionService UserSessionService
}

type UserService interface {
	CreateUser(userEntity user_entity.User) (user.User, error)
	UserByRefreshToken(refreshToken string) (user.User, error)
	UserByEmail(email string) (user.User, error)
}

type UserSessionService interface {
	CreateUserSession(userSessionEntity user_session_entity.UserSession) (user.Session, error)
	DeleteUserSession(userId int64, refreshToken string) error
}

func New(c *config.Config, db *sql.DB, j *jwt.JWT) *Auth {
	return &Auth{
		f:                  "auth",
		c:                  c,
		jwt:                j,
		userService:        user_service.New(db),
		userSessionService: user_session_service.New(db),
	}
}

// LoginHandler godoc
//
//	@Summary		User login
//	@Description	Auth user using email and password
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		profile_entity.LoginRequest		true	"Credentials to auth"
//	@Success		200			{object}	profile_entity.LoginResponse	"Success auth"
//	@Failure		400			{object}	entity.ErrorResponse			"Bad request"
//	@Failure		401			{object}	entity.ErrorResponse			"Unauthorized"
//	@Header			200			{string}	refresh_token					"Set refresh token in cookie to recreate access_token"
//	@Router			/auth/sign-in [post]
func (a *Auth) LoginHandler(c *fiber.Ctx) error {
	const op = "loginHandler"

	controller.SetCommonHeaders(c)

	var r profile_entity.LoginRequest
	err := c.BodyParser(&r)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	if r.Email == "" || r.Password == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(
			&entity.ErrorResponse{Message: controller.MessageLoginOrPasswordInvalid},
		)
	}

	us, err := a.userService.UserByEmail(r.Email)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			logger.Add(a.f, op, err)
		}

		return c.Status(fiber.StatusUnauthorized).JSON(
			&entity.ErrorResponse{Message: controller.MessageLoginOrPasswordInvalid},
		)
	}

	if !password.CheckPassword(r.Password, us.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(
			&entity.ErrorResponse{Message: controller.MessageLoginOrPasswordInvalid},
		)
	}

	accessToken, refreshToken, err := a.tokens(us.Id)
	if err != nil {
		logger.Add(a.f, op, err)

		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	expires := time.Now().Add(time.Hour * time.Duration(a.c.CookieExpires))
	_, err = a.userSessionService.CreateUserSession(user_session_entity.UserSession{
		UserId:       us.Id,
		RefreshToken: refreshToken,
		ExpiredAt:    expires.Format(time.DateTime),
	})
	if err != nil {
		if !errors.Is(err, user_session_service.ErrAlreadyExists) {
			logger.Add(a.f, op, err)
		}

		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	a.setCookie(c, refreshToken, expires)
	c.Status(fiber.StatusOK)

	return c.JSON(&profile_entity.LoginResponse{
		AccessToken: accessToken,
	})
}

// RegisterHandler godoc
//
//	@Summary		User Registration
//	@Description	Register user using email and password
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		profile_entity.RegisterRequest	true	"Credentials to register"
//	@Success		201			{object}	profile_entity.LoginResponse	"User created"
//	@Failure		400			{object}	entity.ErrorResponse			"Bad request"
//	@Failure		409			{object}	entity.ErrorResponse			"User already exists"
//	@Header			200			{string}	refresh_token					"Set refresh token in cookie to recreate access_token"
//	@Router			/auth/sign-up [post]
func (a *Auth) RegisterHandler(c *fiber.Ctx) error {
	const op = "registerHandler"

	controller.SetCommonHeaders(c)

	var r profile_entity.RegisterRequest
	err := c.BodyParser(&r)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	us, err := a.userService.CreateUser(user_entity.User{
		Email:    r.Email,
		Password: r.Password,
	})
	if err != nil {
		if errors.Is(err, user_service.ErrAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(
				&entity.ErrorResponse{Message: controller.MessageUserAlreadyExists},
			)
		}

		logger.Add(a.f, op, err)

		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	accessToken, refreshToken, err := a.tokens(us.Id)
	if err != nil {
		logger.Add(a.f, op, err)

		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	expires := time.Now().Add(time.Hour * time.Duration(a.c.CookieExpires))
	_, err = a.userSessionService.CreateUserSession(user_session_entity.UserSession{
		UserId:       us.Id,
		RefreshToken: refreshToken,
		ExpiredAt:    expires.Format(time.DateTime),
	})
	if err != nil {
		if !errors.Is(err, user_session_service.ErrAlreadyExists) {
			logger.Add(a.f, op, err)
		}

		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	a.setCookie(c, refreshToken, expires)
	c.Status(fiber.StatusCreated)

	return c.JSON(&profile_entity.LoginResponse{
		AccessToken: accessToken,
	})
}

// RefreshHandler godoc
//
//	@Summary		Refresh access_token
//	@Description	Create new access token by refresh_token
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			refresh_token	header		string										true	"Cookie refresh_token"
//	@Success		200				{object}	profile_entity.RefreshAccessTokenResponse	"User email and new access_token"
//	@Failure		401				{object}	entity.ErrorResponse						"Unauthorized"
//	@Router			/auth/refresh-token [post]
func (a *Auth) RefreshHandler(c *fiber.Ctx) error {
	const op = "refreshHandler"

	controller.SetCommonHeaders(c)

	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	us, err := a.userService.UserByRefreshToken(refreshToken)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			logger.Add(a.f, op, err)
		}

		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	accessToken, err := a.jwt.GenerateAccessToken(us.Id)
	if err != nil {
		logger.Add(a.f, op, err)

		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	c.Status(fiber.StatusOK)

	return c.JSON(&profile_entity.RefreshAccessTokenResponse{
		Email:       us.Email.String,
		AccessToken: accessToken,
	})
}

// LogoutHandler godoc
//
//	@Summary		User logout
//	@Description	Sign out
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			refresh_token	header	string	true	"Cookie refresh_token"
//	@Success		200
//	@Failure		401	{object}	entity.ErrorResponse	"Unauthorized"
//	@Router			/auth/sign-out [post]
func (a *Auth) LogoutHandler(c *fiber.Ctx) error {
	const op = "logoutHandler"

	controller.SetCommonHeaders(c)

	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	us, err := a.userService.UserByRefreshToken(refreshToken)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			logger.Add(a.f, op, err)
		}

		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	err = a.userSessionService.DeleteUserSession(us.Id, refreshToken)
	if err != nil {
		logger.Add(a.f, op, err)

		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	// clear cookie
	a.setCookie(c, "", time.Now())
	c.Status(fiber.StatusOK)

	return nil
}

func (a *Auth) setCookie(c *fiber.Ctx, refreshToken string, expires time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HTTPOnly: true,
		Secure:   a.c.CookieSecure,
		SameSite: a.c.CookieSameSite,
		Expires:  expires,
	})
}

func (a *Auth) tokens(userId int64) (string, string, error) {
	const op = "tokens"

	accessToken, err := a.jwt.GenerateAccessToken(userId)
	if err != nil {
		return "", "", logger.Error(a.f, op, err)
	}

	refreshToken, err := a.jwt.GenerateRefreshToken()
	if err != nil {
		return "", "", logger.Error(a.f, op, err)
	}

	return accessToken, refreshToken, nil
}
