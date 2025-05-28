package usersession

import (
	"database/sql"
	"errors"
	"github.com/albakov/go-cloud-file-storage/internal/logger"
	"github.com/albakov/go-cloud-file-storage/internal/storage"
	"github.com/albakov/go-cloud-file-storage/internal/storage/usersession"
	"time"
)

var (
	ErrNotFound       = errors.New("user session not found")
	ErrAlreadyExists  = errors.New("user session already exists")
	ErrSessionExpired = errors.New("user session expired")
)

type Service struct {
	pkg             string
	userSessionRepo Repository
}

type Repository interface {
	ByRefreshToken(refreshToken string) (usersession.Session, error)
	Create(userSession usersession.Session) (usersession.Session, error)
	Delete(userId int64, refreshToken string) error
}

func NewService(db *sql.DB) *Service {
	return &Service{
		pkg:             "usersession.service",
		userSessionRepo: usersession.NewRepository(db),
	}
}

func (s *Service) ValidUserSessionByRefreshToken(refreshToken string) (usersession.Session, error) {
	const op = "ValidUserSessionByRefreshToken"

	us, err := s.userSessionRepo.ByRefreshToken(refreshToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return usersession.Session{}, ErrNotFound
		}

		return usersession.Session{}, logger.Error(s.pkg, op, err)
	}

	expiresAt, err := time.Parse(time.DateTime, us.ExpiredAt)
	if err != nil {
		return usersession.Session{}, err
	}

	if time.Now().After(expiresAt) {
		return usersession.Session{}, ErrSessionExpired
	}

	return us, nil
}

func (s *Service) CreateUserSession(userSessionEntity UserSession) (usersession.Session, error) {
	const op = "CreateUserSession"

	us, err := s.userSessionRepo.Create(usersession.Session{
		UserId:       userSessionEntity.UserId,
		RefreshToken: userSessionEntity.RefreshToken,
		ExpiredAt:    userSessionEntity.ExpiredAt,
	})
	if err != nil {
		if errors.Is(err, storage.ErrDuplicateNotAllowed) {
			return usersession.Session{}, ErrAlreadyExists
		}

		return usersession.Session{}, logger.Error(s.pkg, op, err)
	}

	return us, nil
}

func (s *Service) DeleteUserSession(userId int64, refreshToken string) error {
	const op = "DeleteUserSession"

	err := s.userSessionRepo.Delete(userId, refreshToken)
	if err != nil {
		return logger.Error(s.pkg, op, err)
	}

	return nil
}
