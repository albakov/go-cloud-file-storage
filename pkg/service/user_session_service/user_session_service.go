package user_session_service

import (
	"database/sql"
	"errors"
	"github.com/albakov/go-cloud-file-storage/pkg/logger"
	"github.com/albakov/go-cloud-file-storage/pkg/service/user_session_service/user_session_entity"
	"github.com/albakov/go-cloud-file-storage/pkg/storage"
	"github.com/albakov/go-cloud-file-storage/pkg/storage/entity/user"
	"github.com/albakov/go-cloud-file-storage/pkg/storage/mariadb/user_session_storage"
)

var (
	ErrAlreadyExists = errors.New("user session already exists")
)

type UserSessionService struct {
	f                  string
	userSessionStorage UserSessionStorage
}

type UserSessionStorage interface {
	Create(userSession user.Session) (user.Session, error)
	Delete(userId int64, refreshToken string) error
}

func New(db *sql.DB) *UserSessionService {
	return &UserSessionService{
		f:                  "user_session_service",
		userSessionStorage: user_session_storage.New(db),
	}
}

func (uss *UserSessionService) CreateUserSession(userSessionEntity user_session_entity.UserSession) (user.Session, error) {
	const op = "CreateUserSession"

	userSession, err := uss.userSessionStorage.Create(user.Session{
		UserId:       userSessionEntity.UserId,
		RefreshToken: userSessionEntity.RefreshToken,
		ExpiredAt:    userSessionEntity.ExpiredAt,
	})
	if err != nil {
		if errors.Is(err, storage.ErrDuplicateNotAllowed) {
			return user.Session{}, ErrAlreadyExists
		}

		return user.Session{}, logger.Error(uss.f, op, err)
	}

	return userSession, nil
}

func (uss *UserSessionService) DeleteUserSession(userId int64, refreshToken string) error {
	const op = "DeleteUserSession"

	err := uss.userSessionStorage.Delete(userId, refreshToken)
	if err != nil {
		return logger.Error(uss.f, op, err)
	}

	return nil
}
