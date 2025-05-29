package resource

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/albakov/go-cloud-file-storage/internal/api/controller"
	"github.com/albakov/go-cloud-file-storage/internal/api/entity"
	"github.com/albakov/go-cloud-file-storage/internal/api/entity/resource"
	"github.com/albakov/go-cloud-file-storage/internal/config"
	"github.com/albakov/go-cloud-file-storage/internal/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/minio/minio-go/v7"
	"mime/multipart"
	"path/filepath"
	"strings"
)

type Resource struct {
	pkg       string
	conf      *config.Config
	s3Service S3Service
}

type S3Service interface {
	Object(ctx context.Context, path resource.Path) (*minio.Object, error)
	StoreObject(ctx context.Context, files []*multipart.FileHeader, paths map[string]string, userId int64, path resource.Path) *[]resource.Response
	Delete(ctx context.Context, path resource.Path) error

	Move(ctx context.Context, to, from resource.Path) error
	Search(ctx context.Context, userId int64, query string) *[]resource.Response
	MakeZip(ctx context.Context, path resource.Path) (*bytes.Buffer, error)

	StoreDirectory(ctx context.Context, path resource.Path) (minio.UploadInfo, error)
	PaginateDirectory(ctx context.Context, userId int64, path resource.Path) *[]resource.Response

	AbsPathToObject(userId int64, path string) string
	PathToObjectWithoutPrefix(prefix, path string) string
	ObjectType(filePath string) string
	UserFolderPath(userId int64) string
}

func New(conf *config.Config, s3Service S3Service) *Resource {
	return &Resource{
		pkg:       "resource",
		conf:      conf,
		s3Service: s3Service,
	}
}

// ShowHandler godoc
//
//	@Summary		Show resource
//	@Description	Show resource data
//	@Tags			resource
//	@Accept			json
//	@Produce		json
//	@Param			path			query		string					true	"path=/folder1/folder2/"
//	@Param			Authorization	header		string					true	"Authorization Bearer <ACCESS_TOKEN>"
//	@Success		200				{object}	resource.Response		"Resource data"
//	@Failure		401				{object}	entity.ErrorResponse	"Unauthorized"
//	@Failure		404				{object}	entity.ErrorResponse	"Not found"
//	@Failure		500				{object}	entity.ErrorResponse	"Server error"
//	@Router			/resource [get]
func (res *Resource) ShowHandler(ctx *fiber.Ctx) error {
	const op = "ShowHandler"

	controller.SetCommonHeaders(ctx)

	userId := controller.RequestedUserId(ctx)

	path, err := res.requestedPath(ctx, "path", userId)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(&entity.ErrorResponse{Message: controller.MessageNotFound})
	}

	object, err := res.s3Service.Object(ctx.Context(), path)
	if err != nil {
		logger.Add(res.pkg, op, err)

		return ctx.Status(fiber.StatusNotFound).JSON(&entity.ErrorResponse{Message: controller.MessageNotFound})
	}

	stat, err := object.Stat()
	if err != nil {
		logger.Add(res.pkg, op, err)

		return ctx.Status(fiber.StatusInternalServerError).JSON(
			&entity.ErrorResponse{Message: controller.MessageServerError},
		)
	}

	ctx.Status(fiber.StatusOK)

	return ctx.JSON(&resource.Response{
		Path: res.s3Service.PathToObjectWithoutPrefix(res.s3Service.UserFolderPath(userId), stat.Key),
		Name: filepath.Base(stat.Key),
		Size: stat.Size,
		Type: res.s3Service.ObjectType(stat.Key),
	})
}

// StoreHandler godoc
//
//	@Summary		Store resource
//	@Description	Store resource in the given path
//	@Tags			resource
//	@Accept			json
//	@Produce		json
//	@Param			path			query		string					true	"path=/folder1/folder2/"
//	@Param			paths			formData	string					true	"Must consist json string with paths. Keys are name of resource and values are full path. Example: {'folder':'/folder1/folder/',...}"
//	@Param			files			formData	[]file					true	"Uploading files"
//	@Param			Authorization	header		string					true	"Authorization Bearer <ACCESS_TOKEN>"
//	@Success		201				{object}	[]resource.Response		"Returns list of created resources"
//	@Failure		400				{object}	entity.ErrorResponse	"Bad request"
//	@Failure		401				{object}	entity.ErrorResponse	"Unauthorized"
//	@Router			/resource [post]
func (res *Resource) StoreHandler(ctx *fiber.Ctx) error {
	const op = "StoreHandler"

	controller.SetCommonHeaders(ctx)

	userId := controller.RequestedUserId(ctx)

	path, err := res.requestedPath(ctx, "path", userId)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	pathsJson := ctx.FormValue("paths")
	paths := make(map[string]string)

	err = json.Unmarshal([]byte(pathsJson), &paths)
	if err != nil {
		logger.Add(res.pkg, op, fmt.Errorf("invalid paths JSON: %w", err))

		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		logger.Add(res.pkg, op, err)

		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	files := form.File["files"]

	for _, file := range files {
		if _, ok := paths[file.Filename]; !ok {
			logger.Add(res.pkg, op, err)

			return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
		}
	}

	data := res.s3Service.StoreObject(ctx.Context(), files, paths, userId, path)

	ctx.Status(fiber.StatusCreated)

	return ctx.JSON(data)
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
func (res *Resource) DeleteHandler(ctx *fiber.Ctx) error {
	const op = "DeleteHandler"

	controller.SetCommonHeaders(ctx)

	userId := controller.RequestedUserId(ctx)
	path, err := res.requestedPath(ctx, "path", userId)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	err = res.s3Service.Delete(ctx.Context(), path)
	if err != nil {
		logger.Add(res.pkg, op, err)

		return ctx.Status(fiber.StatusNotFound).JSON(&entity.ErrorResponse{Message: controller.MessageNotFound})
	}

	ctx.Status(fiber.StatusNoContent)

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
func (res *Resource) DownloadHandler(ctx *fiber.Ctx) error {
	const op = "DownloadHandler"

	ctx.Accepts("application/json")
	ctx.Set(fiber.HeaderAccept, "application/json")

	userId := controller.RequestedUserId(ctx)

	path, err := res.requestedPath(ctx, "path", userId)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	// zip all in directory
	if path.IsDirectory {
		buf, err := res.s3Service.MakeZip(ctx.Context(), path)
		if err != nil {
			logger.Add(res.pkg, op, err)

			return ctx.Status(fiber.StatusInternalServerError).JSON(
				&entity.ErrorResponse{Message: controller.MessageServerError},
			)
		}

		ctx.Status(fiber.StatusOK)
		ctx.Set(fiber.HeaderContentType, "application/octet-stream")
		ctx.Set(fiber.HeaderContentDisposition, "attachment; filename=\"archive.zip\"")

		return ctx.Send(buf.Bytes())
	}

	object, err := res.s3Service.Object(ctx.Context(), path)
	if err != nil {
		logger.Add(res.pkg, op, err)

		return ctx.Status(fiber.StatusInternalServerError).JSON(
			&entity.ErrorResponse{Message: controller.MessageServerError},
		)
	}

	stat, err := object.Stat()
	if err != nil {
		logger.Add(res.pkg, op, err)

		return ctx.Status(fiber.StatusInternalServerError).JSON(
			&entity.ErrorResponse{Message: controller.MessageServerError},
		)
	}

	ctx.Set(fiber.HeaderContentType, "application/octet-stream")
	ctx.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(stat.Key)))

	return ctx.SendStream(object)
}

// SearchHandler godoc
//
//	@Summary		Search resource
//	@Description	Search resource by query
//	@Tags			resource
//	@Accept			json
//	@Produce		json
//	@Param			query			query		string					true	"query=file-name"
//	@Param			Authorization	header		string					true	"Authorization Bearer <ACCESS_TOKEN>"
//	@Success		200				{object}	[]resource.Response		"List of resources"
//	@Failure		400				{object}	entity.ErrorResponse	"Bad request"
//	@Failure		401				{object}	entity.ErrorResponse	"Unauthorized"
//	@Router			/resource/search [get]
func (res *Resource) SearchHandler(ctx *fiber.Ctx) error {
	controller.SetCommonHeaders(ctx)

	query := ctx.Query("query", "")
	if query == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	userId := controller.RequestedUserId(ctx)

	data := res.s3Service.Search(ctx.Context(), userId, query)

	ctx.Status(fiber.StatusOK)

	return ctx.JSON(&data)
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
func (res *Resource) MoveHandler(ctx *fiber.Ctx) error {
	const op = "MoveHandler"

	controller.SetCommonHeaders(ctx)

	userId := controller.RequestedUserId(ctx)

	from, err := res.requestedPath(ctx, "from", userId)
	if err != nil || from.CleanPath == "/" {
		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	to, err := res.requestedPath(ctx, "to", userId)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	// prevent create root folder as a child to self
	if strings.HasPrefix(to.CleanPath, from.CleanPath) {
		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	err = res.s3Service.Move(ctx.Context(), to, from)
	if err != nil {
		logger.Add(res.pkg, op, err)

		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	ctx.Status(fiber.StatusNoContent)

	return nil
}

// DirectoryShowHandler godoc
//
//	@Summary		Show resources in the directory
//	@Description	Show resources in the directory
//	@Tags			directory
//	@Accept			json
//	@Produce		json
//	@Param			path			query		string					true	"path=/folder1/folder2/"
//	@Param			Authorization	header		string					true	"Authorization Bearer <ACCESS_TOKEN>"
//	@Success		200				{object}	[]resource.Response		"List of resources"
//	@Failure		401				{object}	entity.ErrorResponse	"Unauthorized"
//	@Failure		404				{object}	entity.ErrorResponse	"Not found"
//	@Router			/directory [get]
func (res *Resource) DirectoryShowHandler(ctx *fiber.Ctx) error {
	controller.SetCommonHeaders(ctx)

	userId := controller.RequestedUserId(ctx)

	path, err := res.requestedPath(ctx, "path", userId)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(&entity.ErrorResponse{Message: controller.MessageNotFound})
	}

	data := res.s3Service.PaginateDirectory(ctx.Context(), userId, path)
	ctx.Status(fiber.StatusOK)

	return ctx.JSON(&data)
}

// DirectoryStoreHandler godoc
//
//	@Summary		Store directory
//	@Description	Create directory in the given path
//	@Tags			directory
//	@Accept			json
//	@Produce		json
//	@Param			path			query		string					true	"path=/folder/new-folder/"
//	@Param			Authorization	header		string					true	"Authorization Bearer <ACCESS_TOKEN>"
//	@Success		201				{object}	resource.Response		"Created resource"
//	@Failure		400				{object}	entity.ErrorResponse	"Bad request"
//	@Failure		401				{object}	entity.ErrorResponse	"Unauthorized"
//	@Router			/directory [post]
func (res *Resource) DirectoryStoreHandler(ctx *fiber.Ctx) error {
	const op = "DirectoryStoreHandler"

	controller.SetCommonHeaders(ctx)

	userId := controller.RequestedUserId(ctx)

	path, err := res.requestedPath(ctx, "path", userId)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	object, err := res.s3Service.StoreDirectory(ctx.Context(), path)
	if err != nil {
		logger.Add(res.pkg, op, err)

		return ctx.Status(fiber.StatusBadRequest).JSON(&entity.ErrorResponse{Message: controller.MessageBadRequest})
	}

	ctx.Status(fiber.StatusCreated)

	return ctx.JSON(&resource.Response{
		Path: fmt.Sprintf("/%s", path.CleanPath),
		Name: filepath.Base(object.Key),
		Size: object.Size,
		Type: "DIRECTORY",
	})
}

func (res *Resource) requestedPath(ctx *fiber.Ctx, key string, userId int64) (resource.Path, error) {
	path := ctx.Query(key, "")
	if path == "" {
		return resource.Path{}, fmt.Errorf("%s is empty", key)
	}

	p := resource.Path{
		OriginalPath: path,
		IsDirectory:  strings.HasSuffix(path, "/"),
	}

	base := res.s3Service.UserFolderPath(userId)
	path = filepath.Clean(filepath.Join(base, path))

	if !strings.HasPrefix(path, base) {
		return resource.Path{}, errors.New("path traversal attempt detected")
	}

	p.CleanPath = path

	return p, nil
}
