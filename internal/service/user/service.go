package user

import (
	"database/sql"
	"errors"
	"github.com/albakov/go-cloud-file-storage/internal/logger"
	"github.com/albakov/go-cloud-file-storage/internal/service/password"
	"github.com/albakov/go-cloud-file-storage/internal/storage"
	"github.com/albakov/go-cloud-file-storage/internal/storage/user"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrAlreadyExists = errors.New("user already exists")
)

type Service struct {
	f        string
	userRepo Repository
}

type Repository interface {
	Create(user user.User) (user.User, error)
	IsExistsByEmail(email string) bool
	ByEmail(email string) (user.User, error)
}

func NewService(db *sql.DB) *Service {
	return &Service{
		f:        "user.service",
		userRepo: user.NewRepository(db),
	}
}

func (s *Service) CreateUser(us User) (user.User, error) {
	const op = "CreateUser"

	if s.userRepo.IsExistsByEmail(us.Email) {
		return user.User{}, ErrAlreadyExists
	}

	hashedPassword, err := password.CreateHashedPassword(us.Password)
	if err != nil {
		return user.User{}, err
	}

	u := user.User{
		Email: sql.NullString{
			String: us.Email,
			Valid:  true,
		},
		Password: hashedPassword,
	}

	u, err = s.userRepo.Create(u)
	if err != nil {
		if errors.Is(err, storage.ErrDuplicateNotAllowed) {
			return user.User{}, ErrAlreadyExists
		}

		return user.User{}, logger.Error(s.f, op, err)
	}

	return u, nil
}

func (s *Service) UserByEmail(email string) (user.User, error) {
	const op = "UserEmail"

	u, err := s.userRepo.ByEmail(email)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return user.User{}, ErrNotFound
		}

		return user.User{}, logger.Error(s.f, op, err)
	}

	return u, nil
}
