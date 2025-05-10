package resource

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/albakov/go-cloud-file-storage/pkg/api/controller"
	"github.com/albakov/go-cloud-file-storage/pkg/api/entity"
	"github.com/albakov/go-cloud-file-storage/pkg/api/entity/resource_entity"
	"github.com/albakov/go-cloud-file-storage/pkg/config"
	"github.com/albakov/go-cloud-file-storage/pkg/logger"
	"github.com/albakov/go-cloud-file-storage/pkg/service/s3/s3_service"
	"github.com/albakov/go-cloud-file-storage/pkg/storage/entity/user"
	"github.com/gofiber/fiber/v2"
	"github.com/minio/minio-go/v7"
	"mime/multipart"
	"path/filepath"
	"strings"
)

type Resource struct {
	f         string
	c         *config.Config
	s3Client  *minio.Client
	s3Service S3Service
}

type UserStorage interface {
	ById(id int64) (user.User, error)
}

type S3Service interface {
	Object(ctx context.Context, path resource_entity.Path) (minio.ObjectInfo, error)
	StoreObject(ctx context.Context, files []*multipart.FileHeader, paths map[string]string, userId int64, path resource_entity.Path) *[]resource_entity.Response
	Delete(ctx context.Context, path resource_entity.Path) error

	Move(ctx context.Context, to, from resource_entity.Path) error
	Search(ctx context.Context, userId int64, query string) *[]resource_entity.Response
	MakeZip(ctx context.Context, path resource_entity.Path) (*bytes.Buffer, error)

	StoreDirectory(ctx context.Context, path resource_entity.Path) (minio.UploadInfo, error)
	PaginateDirectory(ctx context.Context, userId int64, path resource_entity.Path) *[]resource_entity.Response

	AbsPathToObject(userId int64, path string) string
	PathToObjectWithoutPrefix(prefix, path string) string
	ObjectType(filePath string) string
	UserFolderPath(userId int64) string
}

func New(c *config.Config, s3Client *minio.Client) *Resource {
	return &Resource{
		f:         "resource",
		c:         c,
		s3Client:  s3Client,
		s3Service: s3_service.New(s3Client, c),
	}
}

// ShowHandler godoc
//
//	@Summary		Show resource
//	@Description	Show resource data
//	@Tags			resource
//	@Accept			json
//	@Produce		json
//	@Param			path			query		string						true	"path=/folder1/folder2/"
//	@Param			Authorization	header		string						true	"Authorization Bearer <ACCESS_TOKEN>"
//	@Success		200				{object}	resource_entity.Response	"Resource data"
//	@Failure		401				{object}	entity.ErrorResponse		"Unauthorized"
//	@Failure		404				{object}	entity.ErrorResponse		"Not found"
//	@Router			/resource [get]
func (res *Resource) ShowHandler(c *fiber.Ctx) error {
	const op = "ShowHandler"

	controller.SetCommonHeaders(c)

	userId := controller.RequestedUserId(c)
	if userId == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	path, err := res.requestedPath(c, "path", userId)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(&entity.ErrorResponse{Message: controller.MessageNotFound})
	}

	object, err := res.s3Service.Object(c.Context(), path)
	if err != nil {
		logger.Add(res.f, op, err)

		return c.Status(fiber.StatusNotFound).JSON(&entity.ErrorResponse{Message: controller.MessageNotFound})
	}

	c.Status(fiber.StatusOK)

	return c.JSON(&resource_entity.Response{
		Path: res.s3Service.PathToObjectWithoutPrefix(res.s3Service.UserFolderPath(userId), object.Key),
		Name: filepath.Base(object.Key),
		Size: object.Size,
		Type: res.s3Service.ObjectType(object.Key),
	})
}

// StoreHandler godoc
//
//	@Summary		Store resource
//	@Description	Store resource in the given path
//	@Tags			resource
//	@Accept			json
//	@Produce		json
//	@Param			path			query		string						true	"path=/folder1/folder2/"
//	@Param			paths			formData	string						true	"Must consist json string with paths. Keys are name of resource and values are full path. Example: {'folder':'/folder1/folder/',...}"
//	@Param			files			formData	[]file						true	"Uploading files"
//	@Param			Authorization	header		string						true	"Authorization Bearer <ACCESS_TOKEN>"
//	@Success		201				{object}	[]resource_entity.Response	"Returns list of created resources"
//	@Failure		400				{object}	entity.ErrorResponse		"Bad request"
//	@Failure		401				{object}	entity.ErrorResponse		"Unauthorized"
//	@Router			/resource [post]
func (res *Resource) StoreHandler(c *fiber.Ctx) error {
	const op = "StoreHandler"

	controller.SetCommonHeaders(c)

	userId := controller.RequestedUserId(c)
	if userId == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	path, err := res.requestedPath(c, "path", userId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	pathsJson := c.FormValue("paths")
	paths := make(map[string]string)

	err = json.Unmarshal([]byte(pathsJson), &paths)
	if err != nil {
		logger.Add(res.f, op, fmt.Errorf("invalid paths JSON: %w", err))

		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	form, err := c.MultipartForm()
	if err != nil {
		logger.Add(res.f, op, err)

		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	files := form.File["files"]

	for _, file := range files {
		if _, ok := paths[file.Filename]; !ok {
			logger.Add(res.f, op, err)

			return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
		}
	}

	data := res.s3Service.StoreObject(c.Context(), files, paths, userId, path)

	c.Status(fiber.StatusCreated)

	return c.JSON(data)
}

// DeleteHandler godoc
//
//	@Summary		Delete resource
//	@Description	Delete resource in the given path
//	@Tags			resource
//	@Accept			json
//	@Produce		json
//	@Param			path			query		string					true	"path=/folder1/folder2/"
//	@Param			Authorization	header		string					true	"Authorization Bearer <ACCESS_TOKEN>"
//	@Success		204				{object}	nil						"No content"
//	@Failure		400				{object}	entity.ErrorResponse	"Bad request"
//	@Failure		401				{object}	entity.ErrorResponse	"Unauthorized"
//	@Failure		404				{object}	entity.ErrorResponse	"Not found"
//	@Router			/resource [delete]
func (res *Resource) DeleteHandler(c *fiber.Ctx) error {
	const op = "DeleteHandler"

	controller.SetCommonHeaders(c)

	userId := controller.RequestedUserId(c)
	if userId == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	path, err := res.requestedPath(c, "path", userId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	err = res.s3Service.Delete(c.Context(), path)
	if err != nil {
		logger.Add(res.f, op, err)

		return c.Status(fiber.StatusNotFound).JSON(&entity.ErrorResponse{Message: controller.MessageNotFound})
	}

	c.Status(fiber.StatusNoContent)

	return nil
}

// DownloadHandler godoc
//
//	@Summary		Download resource
//	@Description	Download resource from the given path
//	@Tags			resource
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			path			query		string					true	"path=/folder1/folder2/"
//	@Param			Authorization	header		string					true	"Authorization Bearer <ACCESS_TOKEN>"
//	@Success		200				{string}	binary					"If path is a folder, returns zip archive, else - attachment. Content-Type for response is application/octet-stream"
//	@Failure		400				{object}	entity.ErrorResponse	"Bad request"
//	@Failure		401				{object}	entity.ErrorResponse	"Unauthorized"
//	@Failure		404				{object}	entity.ErrorResponse	"Not found"
//	@Failure		500				{object}	entity.ErrorResponse	"Server error"
//	@Router			/resource/download [get]
func (res *Resource) DownloadHandler(c *fiber.Ctx) error {
	const op = "DownloadHandler"

	c.Accepts("application/json")
	c.Set(fiber.HeaderAccept, "application/json")

	userId := controller.RequestedUserId(c)
	if userId == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	path, err := res.requestedPath(c, "path", userId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	// zip all in directory
	if path.IsDirectory {
		buf, err := res.s3Service.MakeZip(c.Context(), path)
		if err != nil {
			logger.Add(res.f, op, err)

			return c.Status(fiber.StatusInternalServerError).JSON(
				&entity.ErrorResponse{Message: controller.MessageServerError},
			)
		}

		c.Status(fiber.StatusOK)
		c.Set(fiber.HeaderContentType, "application/octet-stream")
		c.Set(fiber.HeaderContentDisposition, "attachment; filename=\"archive.zip\"")

		return c.Send(buf.Bytes())
	}

	object, err := res.s3Client.GetObject(c.Context(), res.c.S3Bucket, path.CleanPath, minio.GetObjectOptions{})
	if err != nil {
		logger.Add(res.f, op, err)

		return c.Status(fiber.StatusInternalServerError).JSON(
			&entity.ErrorResponse{Message: controller.MessageServerError},
		)
	}

	stat, err := object.Stat()
	if err != nil {
		logger.Add(res.f, op, err)

		return c.Status(fiber.StatusInternalServerError).JSON(
			&entity.ErrorResponse{Message: controller.MessageServerError},
		)
	}

	c.Set(fiber.HeaderContentType, "application/octet-stream")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(stat.Key)))

	return c.SendStream(object)
}

// SearchHandler godoc
//
//	@Summary		Search resource
//	@Description	Search resource by query
//	@Tags			resource
//	@Accept			json
//	@Produce		json
//	@Param			query			query		string						true	"query=file-name"
//	@Param			Authorization	header		string						true	"Authorization Bearer <ACCESS_TOKEN>"
//	@Success		200				{object}	[]resource_entity.Response	"List of resources"
//	@Failure		400				{object}	entity.ErrorResponse		"Bad request"
//	@Failure		401				{object}	entity.ErrorResponse		"Unauthorized"
//	@Router			/resource/search [get]
func (res *Resource) SearchHandler(c *fiber.Ctx) error {
	controller.SetCommonHeaders(c)

	query := c.Query("query", "")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	userId := controller.RequestedUserId(c)
	if userId == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	data := res.s3Service.Search(c.Context(), userId, query)

	c.Status(fiber.StatusOK)

	return c.JSON(&data)
}

// MoveHandler godoc
//
//	@Summary		Move resource
//	@Description	Move resource $from $to
//	@Tags			resource
//	@Accept			json
//	@Produce		json
//	@Param			from			query		string					true	"from=/folder/file"
//	@Param			to				query		string					true	"to=/another-folder/file"
//	@Param			Authorization	header		string					true	"Authorization Bearer <ACCESS_TOKEN>"
//	@Success		204				{object}	nil						"No content"
//	@Failure		400				{object}	entity.ErrorResponse	"Bad request"
//	@Failure		401				{object}	entity.ErrorResponse	"Unauthorized"
//	@Router			/resource/move [get]
func (res *Resource) MoveHandler(c *fiber.Ctx) error {
	const op = "MoveHandler"

	controller.SetCommonHeaders(c)

	userId := controller.RequestedUserId(c)
	if userId == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	from, err := res.requestedPath(c, "from", userId)
	if err != nil || from.CleanPath == "/" {
		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	to, err := res.requestedPath(c, "to", userId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	// prevent create root folder as a child to self
	if strings.HasPrefix(to.CleanPath, from.CleanPath) {
		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	err = res.s3Service.Move(c.Context(), to, from)
	if err != nil {
		logger.Add(res.f, op, err)

		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	c.Status(fiber.StatusNoContent)

	return nil
}

// DirectoryShowHandler godoc
//
//	@Summary		Show resources in the directory
//	@Description	Show resources in the directory
//	@Tags			directory
//	@Accept			json
//	@Produce		json
//	@Param			path			query		string						true	"path=/folder1/folder2/"
//	@Param			Authorization	header		string						true	"Authorization Bearer <ACCESS_TOKEN>"
//	@Success		200				{object}	[]resource_entity.Response	"List of resources"
//	@Failure		401				{object}	entity.ErrorResponse		"Unauthorized"
//	@Failure		404				{object}	entity.ErrorResponse		"Not found"
//	@Router			/directory [get]
func (res *Resource) DirectoryShowHandler(c *fiber.Ctx) error {
	controller.SetCommonHeaders(c)

	userId := controller.RequestedUserId(c)
	if userId == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	path, err := res.requestedPath(c, "path", userId)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(&entity.ErrorResponse{Message: controller.MessageNotFound})
	}

	data := res.s3Service.PaginateDirectory(c.Context(), userId, path)
	c.Status(fiber.StatusOK)

	return c.JSON(&data)
}

// DirectoryStoreHandler godoc
//
//	@Summary		Store directory
//	@Description	Create directory in the given path
//	@Tags			directory
//	@Accept			json
//	@Produce		json
//	@Param			path			query		string						true	"path=/folder/new-folder/"
//	@Param			Authorization	header		string						true	"Authorization Bearer <ACCESS_TOKEN>"
//	@Success		201				{object}	resource_entity.Response	"Created resource"
//	@Failure		400				{object}	entity.ErrorResponse		"Bad request"
//	@Failure		401				{object}	entity.ErrorResponse		"Unauthorized"
//	@Router			/directory [post]
func (res *Resource) DirectoryStoreHandler(c *fiber.Ctx) error {
	const op = "DirectoryStoreHandler"

	controller.SetCommonHeaders(c)

	userId := controller.RequestedUserId(c)
	if userId == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(&entity.ErrorResponse{Message: controller.MessageUnauthorized})
	}

	path, err := res.requestedPath(c, "path", userId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	object, err := res.s3Service.StoreDirectory(c.Context(), path)
	if err != nil {
		logger.Add(res.f, op, err)

		return c.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	c.Status(fiber.StatusCreated)

	return c.JSON(&resource_entity.Response{
		Path: fmt.Sprintf("/%s", path.CleanPath),
		Name: filepath.Base(object.Key),
		Size: object.Size,
		Type: "DIRECTORY",
	})
}

func (res *Resource) requestedPath(c *fiber.Ctx, key string, userId int64) (resource_entity.Path, error) {
	path := c.Query(key, "")
	if path == "" {
		return resource_entity.Path{}, fmt.Errorf("%s is empty", key)
	}

	p := resource_entity.Path{
		OriginalPath: path,
		IsDirectory:  strings.HasSuffix(path, "/"),
	}

	base := res.s3Service.UserFolderPath(userId)
	path = filepath.Clean(filepath.Join(base, path))

	if !strings.HasPrefix(path, base) {
		return resource_entity.Path{}, errors.New("path traversal attempt detected")
	}

	p.CleanPath = path

	return p, nil
}
