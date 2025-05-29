package main

import (
	"context"
	_ "github.com/albakov/go-cloud-file-storage/docs"
	"github.com/albakov/go-cloud-file-storage/internal/api"
	"github.com/albakov/go-cloud-file-storage/internal/config"
	"github.com/albakov/go-cloud-file-storage/internal/logger"
	"github.com/albakov/go-cloud-file-storage/internal/service/jwt"
	userservice "github.com/albakov/go-cloud-file-storage/internal/service/user"
	usersessionservice "github.com/albakov/go-cloud-file-storage/internal/service/usersession"
	"github.com/albakov/go-cloud-file-storage/internal/storage"
	"github.com/albakov/go-cloud-file-storage/internal/storage/user"
	"github.com/albakov/go-cloud-file-storage/internal/storage/usersession"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//	@title			Cloud File Storage API
//	@version		1.0
//	@description	This is a cloud file storage server.

// @host		localhost:80
// @BasePath	/api
func main() {
	// init config and db connection
	conf := config.MustNew("")
	dbClient := storage.MustNewClient(conf.MysqlDSN)

	// create user service
	userRepo := user.NewRepository(dbClient.DB())
	userService := userservice.NewService(userRepo)

	// create user session service
	userSessionRepo := usersession.NewRepository(dbClient.DB())
	userSessionService := usersessionservice.NewService(userSessionRepo)

	// create jwt service
	jwtService := jwt.NewService(&jwt.Config{Secret: conf.JWTSecret, ExpiresMinutes: conf.JWTExpiresMinutes})

	// create api client
	apiClient := api.MustNewClient(conf, jwtService, userService, userSessionService)
	apiClient.Start()

	// listen for app shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// shutdown api client
	if err := apiClient.Shutdown(); err != nil {
		logger.Add("main", "main", err)
	}

	// shutdown db connection
	if err := dbClient.Shutdown(); err != nil {
		logger.Add("main", "main", err)
	}
}
