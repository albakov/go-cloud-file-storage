package main

import (
	"context"
	_ "github.com/albakov/go-cloud-file-storage/docs"
	"github.com/albakov/go-cloud-file-storage/internal/api"
	"github.com/albakov/go-cloud-file-storage/internal/config"
	"github.com/albakov/go-cloud-file-storage/internal/logger"
	"github.com/albakov/go-cloud-file-storage/internal/storage"
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
	conf := config.MustNew("")
	dbClient := storage.MustNewClient(conf)
	apiClient := api.MustNewClient(conf, dbClient.DB())
	apiClient.Start()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	<-sigs

	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := apiClient.Shutdown(); err != nil {
		logger.Add("main", "main", err)
	}

	if err := dbClient.Shutdown(); err != nil {
		logger.Add("main", "main", err)
	}
}
