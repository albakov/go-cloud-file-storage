package api

import (
	"database/sql"
	"github.com/albakov/go-cloud-file-storage/internal/api/controller/auth"
	"github.com/albakov/go-cloud-file-storage/internal/api/controller/profile"
	"github.com/albakov/go-cloud-file-storage/internal/api/controller/resource"
	"github.com/albakov/go-cloud-file-storage/internal/api/middleware"
	"github.com/albakov/go-cloud-file-storage/internal/config"
	"github.com/albakov/go-cloud-file-storage/internal/service/jwt"
	"github.com/gofiber/swagger"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	_ "github.com/albakov/go-cloud-file-storage/docs"
)

func MustStart(c *config.Config, db *sql.DB) {
	app := fiber.New(fiber.Config{
		BodyLimit: c.ApiFileUploadMaxSize * 1024 * 1024,
	})

	j := jwt.MustNew(c)

	app.Use(cors.New(cors.Config{
		AllowOrigins:     c.CORSAllowOrigins,
		AllowMethods:     c.CORSAllowMethods,
		AllowHeaders:     c.CORSAllowHeaders,
		AllowCredentials: c.CORSAllowCredentials,
	}))

	// middlewares
	m := middleware.New(j)

	// auth
	authCnt := auth.New(c, db, j)
	app.Post("/api/auth/sign-in", authCnt.LoginHandler)
	app.Post("/api/auth/sign-up", authCnt.RegisterHandler)
	app.Post("/api/auth/refresh-token", authCnt.RefreshHandler)
	app.Post("/api/auth/sign-out", authCnt.LogoutHandler)

	// profile
	profileCnt := profile.New(db)
	app.Get("/api/user/me", m.AuthenticatedMiddleware, profileCnt.ShowHandler)

	// resource
	resourceCnt := resource.New(c)
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

	log.Fatal(app.Listen(c.ApiAddr))
}
