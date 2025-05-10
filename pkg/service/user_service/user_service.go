package user_service

import (
	"database/sql"
	"errors"
	"github.com/albakov/go-cloud-file-storage/pkg/logger"
	"github.com/albakov/go-cloud-file-storage/pkg/service/password"
	"github.com/albakov/go-cloud-file-storage/pkg/service/user_service/user_entity"
	"github.com/albakov/go-cloud-file-storage/pkg/storage"
	"github.com/albakov/go-cloud-file-storage/pkg/storage/entity/user"
	"github.com/albakov/go-cloud-file-storage/pkg/storage/mariadb/user_storage"
)

var (
	ErrAlreadyExists = errors.New("user already exists")
)

type UserService struct {
	f           string
	userStorage UserStorage
}

type UserStorage interface {
	Create(user user.User) (user.User, error)
	IsExistsByEmail(email string) bool
	ByEmail(email string) (user.User, error)
	ByRefreshToken(refreshToken string) (user.User, error)
}

func New(db *sql.DB) *UserService {
	return &UserService{
		f:           "user_service",
		userStorage: user_storage.New(db),
	}
}

func (us *UserService) CreateUser(userEntity user_entity.User) (user.User, error) {
	const op = "CreateUser"

	if us.userStorage.IsExistsByEmail(userEntity.Email) {
		return user.User{}, ErrAlreadyExists
	}

	hashedPassword, err := password.CreateHashedPassword(userEntity.Password)
	if err != nil {
		return user.User{}, err
	}

	u := user.User{
		Email: sql.NullString{
			String: userEntity.Email,
			Valid:  true,
		},
		Password: hashedPassword,
	}

	u, err = us.userStorage.Create(u)
	if err != nil {
		if errors.Is(err, storage.ErrDuplicateNotAllowed) {
			return user.User{}, ErrAlreadyExists
		}

		return user.User{}, logger.Error(us.f, op, err)
	}

	return u, nil
}

func (us *UserService) UserByRefreshToken(refreshToken string) (user.User, error) {
	const op = "UserByRefreshToken"

	u, err := us.userStorage.ByRefreshToken(refreshToken)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return user.User{}, storage.ErrNotFound
		}

		return user.User{}, logger.Error(us.f, op, err)
	}

	return u, nil
}

func (us *UserService) UserByEmail(email string) (user.User, error) {
	const op = "UserEmail"

	u, err := us.userStorage.ByEmail(email)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return user.User{}, storage.ErrNotFound
		}

		return user.User{}, logger.Error(us.f, op, err)
	}

	return u, nil
}
