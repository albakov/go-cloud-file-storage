package storage

import (
	"database/sql"
	"fmt"
	"github.com/albakov/go-cloud-file-storage/internal/config"
	"github.com/albakov/go-cloud-file-storage/internal/logger"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Client struct {
	db *sql.DB
}

func MustNewClient(conf *config.Config) *Client {
	db, err := sql.Open("mysql", conf.MysqlDSN)
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(time.Minute * 3)

	if err := db.Ping(); err != nil {
		log.Fatal("database is not response:", err)
	}

	return &Client{db: db}
}
func (cl *Client) DB() *sql.DB {
	return cl.db
}

func (cl *Client) Shutdown() error {
	if err := cl.db.Close(); err != nil {
		return logger.Error("storage.Client", "Shutdown", err)
	}

	fmt.Println("DB Shutdown")

	return nil
}
