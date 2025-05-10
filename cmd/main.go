package main

import (
	"database/sql"
	_ "github.com/albakov/go-cloud-file-storage/docs"
	"github.com/albakov/go-cloud-file-storage/pkg/api"
	"github.com/albakov/go-cloud-file-storage/pkg/config"
	"github.com/albakov/go-cloud-file-storage/pkg/service/s3"
	"github.com/albakov/go-cloud-file-storage/pkg/storage/mariadb"
)

//	@title			Cloud File Storage API
//	@version		1.0
//	@description	This is a cloud file storage server.

// @host		localhost:3001
// @BasePath	/api
func main() {
	c := config.MustNew("")
	db := mariadb.MustNew(c)

	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}(db)

	s3client := s3.New(c)
	api.MustStart(c, db, s3client)
}
