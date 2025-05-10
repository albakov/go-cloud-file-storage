package user_session_storage

import (
	"database/sql"
	"errors"
	"github.com/albakov/go-cloud-file-storage/pkg/storage"
	"github.com/go-sql-driver/mysql"

	"github.com/albakov/go-cloud-file-storage/pkg/logger"
	"github.com/albakov/go-cloud-file-storage/pkg/storage/entity/user"
)

type UserSession struct {
	f  string
	db *sql.DB
}

func New(db *sql.DB) *UserSession {
	return &UserSession{
		f:  "user_session_storage",
		db: db,
	}
}

func (us *UserSession) Create(userSession user.Session) (user.Session, error) {
	const op = "Create"

	stmt, err := us.db.Prepare("INSERT INTO users_sessions (user_id, refresh_token, expires_at) VALUES (?, ?, ?)")
	if err != nil {
		return user.Session{}, logger.Error(us.f, op, err)
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			logger.Add(us.f, op, err)
		}
	}(stmt)

	exec, err := stmt.Exec(userSession.UserId, userSession.RefreshToken, userSession.ExpiredAt)
	if err != nil {
		// check if error is because refresh_token duplicate
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return user.Session{}, storage.ErrDuplicateNotAllowed
		}

		return user.Session{}, logger.Error(us.f, op, err)
	}

	id, err := exec.LastInsertId()
	if err != nil {
		return user.Session{}, logger.Error(us.f, op, err)
	}

	userSession.Id = id

	return userSession, nil
}

func (us *UserSession) Delete(userId int64, refreshToken string) error {
	const op = "Delete"

	stmt, err := us.db.Prepare("DELETE FROM users_sessions WHERE user_id = ? AND refresh_token = ?")
	if err != nil {
		return logger.Error(us.f, op, err)
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			logger.Add(us.f, op, err)
		}
	}(stmt)

	_, err = stmt.Exec(userId, refreshToken)
	if err != nil {
		return logger.Error(us.f, op, err)
	}

	return nil
}
