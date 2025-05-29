package auth

import (
	"errors"
	"github.com/albakov/go-cloud-file-storage/internal/api/controller"
	"github.com/albakov/go-cloud-file-storage/internal/api/entity"
	"github.com/albakov/go-cloud-file-storage/internal/api/entity/profile"
	"github.com/albakov/go-cloud-file-storage/internal/config"
	"github.com/albakov/go-cloud-file-storage/internal/logger"
	"github.com/albakov/go-cloud-file-storage/internal/service/password"
	userservice "github.com/albakov/go-cloud-file-storage/internal/service/user"
	usersessionservice "github.com/albakov/go-cloud-file-storage/internal/service/usersession"
	"github.com/albakov/go-cloud-file-storage/internal/storage/user"
	"github.com/albakov/go-cloud-file-storage/internal/storage/usersession"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Auth struct {
	pkg                string
	conf               *config.Config
	authService        AuthService
	userService        UserService
	userSessionService UserSessionService
}

type AuthService interface {
	GenerateAccessToken(userId int64) (string, error)
	GenerateRefreshToken() (string, error)
}

type UserService interface {
	CreateUser(userEntity userservice.User) (user.User, error)
	UserByEmail(email string) (user.User, error)
}

type UserSessionService interface {
	ValidUserSessionByRefreshToken(refreshToken string) (usersession.Session, error)
	CreateUserSession(userSessionEntity usersessionservice.UserSession) (usersession.Session, error)
	DeleteUserSession(userId int64, refreshToken string) error
}

func New(
	conf *config.Config,
	authService AuthService,
	userService UserService,
	userSessionService UserSessionService,
) *Auth {
	return &Auth{
		pkg:                "auth",
		conf:               conf,
		authService:        authService,
		userService:        userService,
		userSessionService: userSessionService,
	}
}

// LoginHandler godoc
//
//	@Summary		User login
//	@Description	Auth user using email and password
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			credentials	body		profile.LoginRequest	true	"Credentials to auth"
//	@Success		200			{object}	profile.LoginResponse	"Success auth"
//	@Failure		400			{object}	entity.ErrorResponse	"Bad request"
//	@Failure		401			{object}	entity.ErrorResponse	"Unauthorized"
//	@Header			200			{string}	refresh_token			"Set refresh token in cookie to recreate access_token"
//	@Router			/auth/sign-in [post]
func (a *Auth) LoginHandler(ctx *fiber.Ctx) error {
	const op = "loginHandler"

	controller.SetCommonHeaders(ctx)
	r := controller.RequestedLogin(ctx)

	us, err := a.userService.UserByEmail(r.Email)
	if err != nil {
		if !errors.Is(err, userservice.ErrNotFound) {
			logger.Add(a.pkg, op, err)
		}

		return ctx.Status(fiber.StatusUnauthorized).JSON(
			&entity.ErrorResponse{Message: controller.MessageLoginOrPasswordInvalid},
		)
	}

	if !password.CheckPassword(r.Password, us.Password) {
		return ctx.Status(fiber.StatusUnauthorized).JSON(
			&entity.ErrorResponse{Message: controller.MessageLoginOrPasswordInvalid},
		)
	}

	accessToken, refreshToken, err := a.tokens(us.Id)
	if err != nil {
		logger.Add(a.pkg, op, err)

		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	expires := time.Now().Add(time.Hour * time.Duration(a.conf.CookieExpires))
	_, err = a.userSessionService.CreateUserSession(usersessionservice.UserSession{
		UserId:       us.Id,
		RefreshToken: refreshToken,
		ExpiredAt:    expires.Format(time.DateTime),
	})
	if err != nil {
		if !errors.Is(err, usersessionservice.ErrAlreadyExists) {
			logger.Add(a.pkg, op, err)
		}

		return ctx.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	a.setCookie(ctx, refreshToken, expires)
	ctx.Status(fiber.StatusOK)

	return ctx.JSON(&profile.LoginResponse{
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
//	@Param			credentials	body		profile.RegisterRequest	true	"Credentials to register"
//	@Success		201			{object}	profile.LoginResponse	"User created"
//	@Failure		400			{object}	entity.ErrorResponse	"Bad request"
//	@Failure		409			{object}	entity.ErrorResponse	"User already exists"
//	@Header			200			{string}	refresh_token			"Set refresh token in cookie to recreate access_token"
//	@Router			/auth/sign-up [post]
func (a *Auth) RegisterHandler(ctx *fiber.Ctx) error {
	const op = "registerHandler"

	controller.SetCommonHeaders(ctx)
	r := controller.RequestedLogin(ctx)

	us, err := a.userService.CreateUser(userservice.User{
		Email:    r.Email,
		Password: r.Password,
	})
	if err != nil {
		if errors.Is(err, userservice.ErrAlreadyExists) {
			return ctx.Status(fiber.StatusConflict).JSON(
				&entity.ErrorResponse{Message: controller.MessageUserAlreadyExists},
			)
		}

		logger.Add(a.pkg, op, err)

		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	accessToken, refreshToken, err := a.tokens(us.Id)
	if err != nil {
		logger.Add(a.pkg, op, err)

		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	expires := time.Now().Add(time.Hour * time.Duration(a.conf.CookieExpires))
	_, err = a.userSessionService.CreateUserSession(usersessionservice.UserSession{
		UserId:       us.Id,
		RefreshToken: refreshToken,
		ExpiredAt:    expires.Format(time.DateTime),
	})
	if err != nil {
		if !errors.Is(err, usersessionservice.ErrAlreadyExists) {
			logger.Add(a.pkg, op, err)
		}

		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	a.setCookie(ctx, refreshToken, expires)
	ctx.Status(fiber.StatusCreated)

	return ctx.JSON(&profile.LoginResponse{
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
//	@Param			refresh_token	header		string								true	"Cookie refresh_token"
//	@Success		200				{object}	profile.RefreshAccessTokenResponse	"New access_token"
//	@Failure		401				{object}	entity.ErrorResponse				"Unauthorized"
//	@Router			/auth/refresh-token [post]
func (a *Auth) RefreshHandler(ctx *fiber.Ctx) error {
	const op = "refreshHandler"

	controller.SetCommonHeaders(ctx)

	refreshToken := ctx.Cookies("refresh_token")
	if refreshToken == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	us, err := a.userSessionService.ValidUserSessionByRefreshToken(refreshToken)
	if err != nil {
		if !errors.Is(err, usersessionservice.ErrNotFound) && !errors.Is(err, usersessionservice.ErrSessionExpired) {
			logger.Add(a.pkg, op, err)
		}

		return ctx.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	accessToken, err := a.authService.GenerateAccessToken(us.UserId)
	if err != nil {
		logger.Add(a.pkg, op, err)

		return ctx.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	ctx.Status(fiber.StatusOK)

	return ctx.JSON(&profile.RefreshAccessTokenResponse{
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
func (a *Auth) LogoutHandler(ctx *fiber.Ctx) error {
	const op = "logoutHandler"

	controller.SetCommonHeaders(ctx)

	refreshToken := ctx.Cookies("refresh_token")
	if refreshToken == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	us, err := a.userSessionService.ValidUserSessionByRefreshToken(refreshToken)
	if err != nil {
		if !errors.Is(err, usersessionservice.ErrNotFound) && !errors.Is(err, usersessionservice.ErrSessionExpired) {
			logger.Add(a.pkg, op, err)
		}

		return ctx.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	err = a.userSessionService.DeleteUserSession(us.UserId, refreshToken)
	if err != nil {
		logger.Add(a.pkg, op, err)

		return ctx.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	// clear cookie
	a.setCookie(ctx, "", time.Now())
	ctx.Status(fiber.StatusOK)

	return nil
}

func (a *Auth) setCookie(ctx *fiber.Ctx, refreshToken string, expires time.Time) {
	ctx.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HTTPOnly: true,
		Secure:   a.conf.CookieSecure,
		SameSite: a.conf.CookieSameSite,
		Expires:  expires,
	})
}

func (a *Auth) tokens(userId int64) (string, string, error) {
	const op = "tokens"

	accessToken, err := a.authService.GenerateAccessToken(userId)
	if err != nil {
		return "", "", logger.Error(a.pkg, op, err)
	}

	refreshToken, err := a.authService.GenerateRefreshToken()
	if err != nil {
		return "", "", logger.Error(a.pkg, op, err)
	}

	return accessToken, refreshToken, nil
}
