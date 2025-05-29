package api

import (
	"errors"
	"fmt"
	_ "github.com/albakov/go-cloud-file-storage/docs"
	"github.com/albakov/go-cloud-file-storage/internal/api/controller/auth"
	"github.com/albakov/go-cloud-file-storage/internal/api/controller/profile"
	"github.com/albakov/go-cloud-file-storage/internal/api/controller/resource"
	"github.com/albakov/go-cloud-file-storage/internal/api/middleware/authenticated"
	"github.com/albakov/go-cloud-file-storage/internal/api/middleware/validation"
	"github.com/albakov/go-cloud-file-storage/internal/config"
	"github.com/albakov/go-cloud-file-storage/internal/logger"
	"github.com/albakov/go-cloud-file-storage/internal/service/jwt"
	"github.com/albakov/go-cloud-file-storage/internal/service/s3"
	userservice "github.com/albakov/go-cloud-file-storage/internal/service/user"
	usersessionservice "github.com/albakov/go-cloud-file-storage/internal/service/usersession"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
	"log"
	"net/http"
)

type Client struct {
	app  *fiber.App
	conf *config.Config
}

func MustNewClient(
	conf *config.Config,
	jwtService *jwt.Service,
	userService *userservice.Service,
	userSessionService *usersessionservice.Service,
) *Client {
	app := fiber.New(fiber.Config{
		BodyLimit: conf.ApiFileUploadMaxSize * 1024 * 1024,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins:     conf.CORSAllowOrigins,
		AllowMethods:     conf.CORSAllowMethods,
		AllowHeaders:     conf.CORSAllowHeaders,
		AllowCredentials: conf.CORSAllowCredentials,
	}))

	// auth
	authCnt := auth.New(conf, jwtService, userService, userSessionService)

	app.Post("/api/auth/sign-in", validation.EmailAndPasswordValidation, authCnt.LoginHandler)
	app.Post("/api/auth/sign-up", validation.EmailAndPasswordValidation, authCnt.RegisterHandler)

	app.Post("/api/auth/refresh-token", authCnt.RefreshHandler)
	app.Post("/api/auth/sign-out", authCnt.LogoutHandler)

	authMiddleware := authenticated.New(jwtService)

	// profile
	profileCnt := profile.New(userService)
	app.Get("/api/user/me", authMiddleware.Authenticated, profileCnt.ShowHandler)

	s3Client := s3.NewClient(conf)
	s3Service := s3.NewService(s3Client, conf.S3Bucket)

	// resource
	resourceCnt := resource.New(conf, s3Service)

	resourceGroup := app.Group("/api/resource")
	resourceGroup.Use(authMiddleware.Authenticated)
	resourceGroup.Get("/", resourceCnt.ShowHandler)
	resourceGroup.Post("/", resourceCnt.StoreHandler)
	resourceGroup.Delete("/", resourceCnt.DeleteHandler)
	resourceGroup.Get("/move", resourceCnt.MoveHandler)
	resourceGroup.Get("/download", resourceCnt.DownloadHandler)
	resourceGroup.Get("/search", resourceCnt.SearchHandler)

	directoryGroup := app.Group("/api/directory")
	directoryGroup.Use(authMiddleware.Authenticated)
	directoryGroup.Get("/", resourceCnt.DirectoryShowHandler)
	directoryGroup.Post("/", resourceCnt.DirectoryStoreHandler)

	// swagger
	app.Get("/swagger/*", swagger.HandlerDefault)

	return &Client{app: app, conf: conf}
}

func (cl *Client) Start() {
	go func() {
		err := cl.app.Listen(cl.conf.ApiAddr)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
}

func (cl *Client) Shutdown() error {
	if err := cl.app.Shutdown(); err != nil {
		return logger.Error("api.Client", "Shutdown", err)
	}

	fmt.Println("API Server Shutdown")

	return nil
}
