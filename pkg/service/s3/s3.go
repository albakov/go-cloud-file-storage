package s3

import (
	"github.com/albakov/go-cloud-file-storage/pkg/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"log"
)

func New(c *config.Config) *minio.Client {
	minioClient, err := minio.New(c.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.S3AccessKey, c.S3SecretAccess, ""),
		Secure: c.S3UseSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	return minioClient
}
