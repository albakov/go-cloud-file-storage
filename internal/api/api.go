package api

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/albakov/go-cloud-file-storage/docs"
	"github.com/albakov/go-cloud-file-storage/internal/api/controller/auth"
	"github.com/albakov/go-cloud-file-storage/internal/api/controller/profile"
	"github.com/albakov/go-cloud-file-storage/internal/api/controller/resource"
	"github.com/albakov/go-cloud-file-storage/internal/api/middleware"
	"github.com/albakov/go-cloud-file-storage/internal/config"
	"github.com/albakov/go-cloud-file-storage/internal/logger"
	"github.com/albakov/go-cloud-file-storage/internal/service/jwt"
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

func MustNewClient(conf *config.Config, db *sql.DB) *Client {
	app := fiber.New(fiber.Config{
		BodyLimit: conf.ApiFileUploadMaxSize * 1024 * 1024,
	})

	j := jwt.MustNew(conf)

	app.Use(cors.New(cors.Config{
		AllowOrigins:     conf.CORSAllowOrigins,
		AllowMethods:     conf.CORSAllowMethods,
		AllowHeaders:     conf.CORSAllowHeaders,
		AllowCredentials: conf.CORSAllowCredentials,
	}))

	// middlewares
	m := middleware.New(j)

	// auth
	authCnt := auth.New(conf, db, j)
	app.Post("/api/auth/sign-in", authCnt.LoginHandler)
	app.Post("/api/auth/sign-up", authCnt.RegisterHandler)
	app.Post("/api/auth/refresh-token", authCnt.RefreshHandler)
	app.Post("/api/auth/sign-out", authCnt.LogoutHandler)

	// profile
	profileCnt := profile.New(db)
	app.Get("/api/user/me", m.AuthenticatedMiddleware, profileCnt.ShowHandler)

	// resource
	resourceCnt := resource.New(conf)
	app.Get("/api/resource", m.AuthenticatedMiddleware, resourceCnt.ShowHandler)
	app.Post("/api/resource", m.AuthenticatedMiddleware, resourceCnt.StoreHandler)
	app.Delete("/api/resource", m.AuthenticatedMiddleware, resourceCnt.DeleteHandler)
	app.Get("/api/resource/move", m.AuthenticatedMiddleware, resourceCnt.MoveHandler)
	app.Get("/api/resource/download", m.AuthenticatedMiddleware, resourceCnt.DownloadHandler)
	app.Get("/api/resource/search", m.AuthenticatedMiddleware, resourceCnt.SearchHandler)
	app.Get("/api/directory", m.AuthenticatedMiddleware, resourceCnt.DirectoryShowHandler)
	app.Post("/api/directory", m.AuthenticatedMiddleware, resourceCnt.DirectoryStoreHandler)

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
