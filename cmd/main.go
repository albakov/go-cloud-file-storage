package main

import (
	"database/sql"
	_ "github.com/albakov/go-cloud-file-storage/docs"
	"github.com/albakov/go-cloud-file-storage/internal/api"
	"github.com/albakov/go-cloud-file-storage/internal/config"
	"github.com/albakov/go-cloud-file-storage/internal/storage"
)

//	@title			Cloud File Storage API
//	@version		1.0
//	@description	This is a cloud file storage server.

// @host		localhost:80
// @BasePath	/api
func main() {
	c := config.MustNew("")
	db := storage.MustNewClient(c)

	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}(db)

	api.MustStart(c, db)
}
