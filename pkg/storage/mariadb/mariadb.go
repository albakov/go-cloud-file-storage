package mariadb

import (
	"database/sql"
	"log"
	"time"

	"github.com/albakov/go-cloud-file-storage/pkg/config"
	_ "github.com/go-sql-driver/mysql"
)

func MustNew(cf *config.Config) *sql.DB {
	db, err := sql.Open("mysql", cf.MysqlDSN)
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(time.Minute * 3)

	if err := db.Ping(); err != nil {
		log.Fatal("database is not response:", err)
	}

	return db
}
