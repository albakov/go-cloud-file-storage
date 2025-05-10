package config

import (
	"errors"
	"fmt"
	"github.com/albakov/go-cloud-file-storage/pkg/logger"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"log"
)

type Config struct {
	MysqlDSN string `mapstructure:"MYSQL_DSN"`

	ApiAddr              string `mapstructure:"API_ADDR"`
	ApiFileUploadMaxSize int    `mapstructure:"API_FILE_UPLOAD_MAX_SIZE"`

	JWTSecret         string `mapstructure:"JWT_SECRET"`
	JWTExpiresMinutes int64  `mapstructure:"JWT_EXPIRES_MINUTES"`

	CookieSecure   bool   `mapstructure:"COOKIE_SECURE"`
	CookieSameSite string `mapstructure:"COOKIE_SAME_SITE"`
	CookieExpires  int64  `mapstructure:"COOKIE_EXPIRES"`

	CORSAllowOrigins     string `mapstructure:"CORS_ALLOW_ORIGINS"`
	CORSAllowMethods     string `mapstructure:"CORS_ALLOW_METHODS"`
	CORSAllowHeaders     string `mapstructure:"CORS_ALLOW_HEADERS"`
	CORSAllowCredentials bool   `mapstructure:"CORS_ALLOW_CREDENTIALS"`

	S3Endpoint     string `mapstructure:"MINIO_ENDPOINT"`
	S3AccessKey    string `mapstructure:"MINIO_ACCESS_KEY"`
	S3SecretAccess string `mapstructure:"MINIO_SECRET_KEY"`
	S3Bucket       string `mapstructure:"MINIO_BUCKET"`
	S3UseSSL       bool   `mapstructure:"MINIO_USE_SSL"`
	S3Paginate     int    `mapstructure:"MINIO_FILES_PAGINATE"`
}

const f = "config"

func MustNew(envPath string) *Config {
	const op = "MustNew"

	var config Config

	if envPath == "" {
		envPath = envFileFromCommandLine()
	}

	viper.SetConfigFile(envPath)
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		logger.Add(f, op, fmt.Errorf("error while reading config file: %v", err))
	}

	if err := viper.Unmarshal(&config); err != nil {
		log.Fatal(err)
	}

	return &config
}

func envFileFromCommandLine() string {
	const op = "envFileFromCommandLine"

	pflag.String("env-file", "", "Path to env file. Example: --env-file env.dev")
	pflag.Parse()

	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		logger.Add(f, op, err)

		return ""
	}

	envFile := viper.GetString("env-file")
	if envFile == "" {
		logger.Add(f, op, errors.New("flag --env-file is empty"))

		return ""
	}

	return envFile
}
