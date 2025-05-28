package s3

import (
	"github.com/albakov/go-cloud-file-storage/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"log"
)

func NewClient(conf *config.Config) *minio.Client {
	minioClient, err := minio.New(conf.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(conf.S3AccessKey, conf.S3SecretAccess, ""),
		Secure: conf.S3UseSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	return minioClient
}
