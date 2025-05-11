package storage

import (
	"database/sql"
	"github.com/albakov/go-cloud-file-storage/internal/config"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func MustNewClient(cf *config.Config) *sql.DB {
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
