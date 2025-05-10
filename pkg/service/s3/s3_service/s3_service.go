package s3_service

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"github.com/albakov/go-cloud-file-storage/pkg/api/entity/resource_entity"
	"github.com/albakov/go-cloud-file-storage/pkg/config"
	"github.com/albakov/go-cloud-file-storage/pkg/logger"
	"github.com/minio/minio-go/v7"
	"io"
	"mime/multipart"
	"path/filepath"
	"slices"
	"strings"
)

type S3Service struct {
	f        string
	s3Client *minio.Client
	c        *config.Config
}

func New(s3Client *minio.Client, c *config.Config) *S3Service {
	return &S3Service{
		f:        "s3_service",
		s3Client: s3Client,
		c:        c,
	}
}

func (s *S3Service) Object(ctx context.Context, path resource_entity.Path) (minio.ObjectInfo, error) {
	const op = "Object"

	object, err := s.s3Client.StatObject(
		ctx,
		s.c.S3Bucket,
		path.CleanPath,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return minio.ObjectInfo{}, logger.Error(s.f, op, err)
	}

	return object, nil
}

func (s *S3Service) StoreObject(
	ctx context.Context,
	files []*multipart.FileHeader,
	paths map[string]string,
	userId int64,
	path resource_entity.Path,
) *[]resource_entity.Response {
	prefix := s.UserFolderPath(userId)

	opts := minio.PutObjectOptions{}
	data := []resource_entity.Response{}

	for _, fileHeader := range files {
		s.uploadFile(ctx, &data, fileHeader, path, prefix, paths, opts)
	}

	return &data
}

func (s *S3Service) Delete(ctx context.Context, path resource_entity.Path) error {
	const op = "Delete"

	// if the resource is a directory - remove all data inside
	if path.IsDirectory {
		s.deleteRecursive(ctx, path.CleanPathWithTailingSlash())

		return nil
	}

	err := s.deleteObject(ctx, path.CleanPath, s.removeOptions())
	if err != nil {
		return logger.Error(s.f, op, err)
	}

	return nil
}

func (s *S3Service) Search(ctx context.Context, userId int64, query string) *[]resource_entity.Response {
	const op = "Search"

	path := fmt.Sprintf("%s/", s.AbsPathToObject(userId, ""))
	opts := minio.ListObjectsOptions{
		Prefix:     path,
		StartAfter: path,
		Recursive:  true,
	}

	data := []resource_entity.Response{}
	prefix := s.UserFolderPath(userId)

	for v := range s.s3Client.ListObjects(ctx, s.c.S3Bucket, opts) {
		if v.Err != nil {
			logger.Add(s.f, op, v.Err)

			continue
		}

		if strings.Contains(v.Key, query) {
			data = append(data, resource_entity.Response{
				Path: s.PathToObjectWithoutPrefix(v.Key, prefix),
				Name: filepath.Base(v.Key),
				Size: v.Size,
				Type: s.ObjectType(v.Key),
			})
		}
	}

	return &data
}

func (s *S3Service) MakeZip(ctx context.Context, path resource_entity.Path) (*bytes.Buffer, error) {
	const op = "MakeZip"

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	opts := minio.ListObjectsOptions{
		Prefix:    path.CleanPathWithTailingSlash(),
		Recursive: true,
	}

	for v := range s.s3Client.ListObjects(ctx, s.c.S3Bucket, opts) {
		err := s.putObjectInZip(ctx, v, zipWriter, path.CleanPath)
		if err != nil {
			return nil, logger.Error(s.f, op, err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, logger.Error(s.f, op, err)
	}

	return buf, nil
}

func (s *S3Service) Move(ctx context.Context, to, from resource_entity.Path) error {
	const op = "Move"

	// if "from" is a directory, move all items inside "to"
	if from.IsDirectory {
		fromPath := from.CleanPathWithTailingSlash() // user-N-files/from/

		err := s.copyRecursive(ctx, to.CleanPathWithTailingSlash(), fromPath)
		if err != nil {
			return logger.Error(s.f, op, err)
		}

		s.deleteRecursive(ctx, fromPath)

		return nil
	}

	_, err := s.s3Client.CopyObject(
		ctx,
		minio.CopyDestOptions{
			Bucket: s.c.S3Bucket,
			Object: filepath.Join(to.CleanPathDirName(), filepath.Base(to.CleanPath)),
		},
		minio.CopySrcOptions{
			Bucket: s.c.S3Bucket,
			Object: from.CleanPath,
		},
	)
	if err != nil {
		return logger.Error(s.f, op, err)
	}

	err = s.deleteObject(ctx, from.CleanPath, s.removeOptions())
	if err != nil {
		return logger.Error(s.f, op, err)
	}

	return nil
}

func (s *S3Service) StoreDirectory(ctx context.Context, path resource_entity.Path) (minio.UploadInfo, error) {
	const op = "StoreDirectory"

	object, err := s.s3Client.PutObject(
		ctx,
		s.c.S3Bucket,
		path.CleanPathWithTailingSlash(),
		nil,
		0,
		minio.PutObjectOptions{},
	)
	if err != nil {
		return minio.UploadInfo{}, logger.Error(s.f, op, err)
	}

	return object, nil
}

func (s *S3Service) PaginateDirectory(ctx context.Context, userId int64, path resource_entity.Path) *[]resource_entity.Response {
	const op = "PaginateDirectory"

	prefix := s.UserFolderPath(userId)
	pathToObject := path.CleanPathWithTailingSlash()

	objects := s.s3Client.ListObjects(ctx, s.c.S3Bucket, minio.ListObjectsOptions{
		Prefix:     pathToObject,
		StartAfter: pathToObject,
	})

	data := []resource_entity.Response{}

	for v := range objects {
		if v.Err != nil {
			logger.Add(s.f, op, v.Err)

			continue
		}

		data = append(data, resource_entity.Response{
			Path: s.PathToObjectWithoutPrefix(v.Key, prefix),
			Name: filepath.Base(v.Key),
			Size: v.Size,
			Type: s.ObjectType(v.Key),
		})
	}

	s.sortObjectsList(&data)

	return &data
}

// AbsPathToObject returns the path to the object with the suffix: "user-USER_ID-files/"
func (s *S3Service) AbsPathToObject(userId int64, path string) string {
	return filepath.Join(s.UserFolderPath(userId), path)
}

// UserFolderPath returns the user's root folder path
func (s *S3Service) UserFolderPath(userId int64) string {
	return fmt.Sprintf("user-%d-files", userId)
}

// PathToObjectWithoutPrefix returns the path without the prefix: "user-USER_ID-files"
func (s *S3Service) PathToObjectWithoutPrefix(path, prefix string) string {
	pathToFile, _ := strings.CutPrefix(path, prefix)

	return pathToFile
}

// ObjectType returns object type FILE or DIRECTORY
func (s *S3Service) ObjectType(filePath string) string {
	if strings.HasSuffix(filePath, "/") {
		return "DIRECTORY"
	}

	return "FILE"
}

func (s *S3Service) uploadFile(
	ctx context.Context,
	data *[]resource_entity.Response,
	file *multipart.FileHeader,
	path resource_entity.Path,
	prefix string,
	paths map[string]string,
	opts minio.PutObjectOptions,
) {
	const op = "uploadFile"

	fileData, err := file.Open()
	if err != nil {
		logger.Add(s.f, op, err)

		return
	}
	defer func(fileData multipart.File) {
		err := fileData.Close()
		if err != nil {
			logger.Add(s.f, op, err)
		}
	}(fileData)

	object, err := s.s3Client.PutObject(
		ctx,
		s.c.S3Bucket,
		filepath.Join(path.CleanPath, paths[file.Filename]),
		fileData,
		file.Size,
		opts,
	)
	if err != nil {
		logger.Add(s.f, op, err)

		return
	}

	*data = append(*data, resource_entity.Response{
		Path: s.PathToObjectWithoutPrefix(object.Key, prefix),
		Name: filepath.Base(object.Key),
		Size: object.Size,
		Type: s.ObjectType(object.Key),
	})
}

func (s *S3Service) deleteRecursive(ctx context.Context, path string) {
	const op = "deleteRecursive"

	opts := minio.ListObjectsOptions{
		Prefix:    path,
		Recursive: true,
	}

	removeOpts := s.removeOptions()

	for object := range s.s3Client.ListObjects(ctx, s.c.S3Bucket, opts) {
		if object.Err != nil {
			logger.Add(s.f, op, object.Err)

			continue
		}

		err := s.deleteObject(ctx, object.Key, removeOpts)
		if err != nil {
			logger.Add(s.f, op, object.Err)

			continue
		}
	}
}

func (s *S3Service) putObjectInZip(ctx context.Context, v minio.ObjectInfo, zipWriter *zip.Writer, prefix string) error {
	const op = "putObjectInZip"

	if v.Err != nil {
		return logger.Error(s.f, op, v.Err)
	}

	// skip directory
	if strings.HasSuffix(v.Key, "/") {
		return nil
	}

	entry, err := zipWriter.Create(s.PathToObjectWithoutPrefix(v.Key, prefix))
	if err != nil {
		return logger.Error(s.f, op, v.Err)
	}

	obj, err := s.s3Client.GetObject(ctx, s.c.S3Bucket, v.Key, minio.GetObjectOptions{})
	if err != nil {
		return logger.Error(s.f, op, v.Err)
	}

	if _, err := io.Copy(entry, obj); err != nil {
		return logger.Error(s.f, op, v.Err)
	}

	if err := obj.Close(); err != nil {
		return logger.Error(s.f, op, v.Err)
	}

	return nil
}

func (s *S3Service) sortObjectsList(data *[]resource_entity.Response) {
	slices.SortFunc(*data, func(a, b resource_entity.Response) int {
		if a.Type == b.Type {
			return 0
		}

		if a.Type == "DIRECTORY" {
			return -1
		}

		return 1
	})
}

func (s *S3Service) copyRecursive(ctx context.Context, to, from string) error {
	const op = "copyRecursive"

	opts := minio.ListObjectsOptions{
		Prefix:     from,
		StartAfter: from,
		Recursive:  true,
	}

	isChanged := false

	for v := range s.s3Client.ListObjects(ctx, s.c.S3Bucket, opts) {
		if v.Err != nil {
			return logger.Error(s.f, op, v.Err)
		}

		isChanged = true
		copyTo := filepath.Join(to, filepath.Base(v.Key))

		// if is a directory, copy resource as directory
		if strings.HasSuffix(v.Key, "/") {
			copyTo = fmt.Sprintf("%s/", copyTo)
		}

		_, err := s.s3Client.CopyObject(
			ctx,
			minio.CopyDestOptions{
				Bucket: s.c.S3Bucket,
				Object: copyTo,
			},
			minio.CopySrcOptions{
				Bucket: s.c.S3Bucket,
				Object: v.Key,
			},
		)
		if err != nil {
			return logger.Error(s.f, op, err)
		}
	}

	// if trying to rename empty folder
	if !isChanged {
		_, err := s.s3Client.CopyObject(
			ctx,
			minio.CopyDestOptions{
				Bucket: s.c.S3Bucket,
				Object: to,
			},
			minio.CopySrcOptions{
				Bucket: s.c.S3Bucket,
				Object: from,
			},
		)
		if err != nil {
			return logger.Error(s.f, op, err)
		}
	}

	return nil
}

func (s *S3Service) deleteObject(ctx context.Context, path string, removeOpts *minio.RemoveObjectOptions) error {
	const op = "deleteObject"

	err := s.s3Client.RemoveObject(ctx, s.c.S3Bucket, path, *removeOpts)
	if err != nil {
		return logger.Error(s.f, op, err)
	}

	return nil
}

func (s *S3Service) removeOptions() *minio.RemoveObjectOptions {
	return &minio.RemoveObjectOptions{
		ForceDelete:      true,
		GovernanceBypass: true,
	}
}
