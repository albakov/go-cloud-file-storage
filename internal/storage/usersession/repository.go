package usersession

import (
	"database/sql"
	"errors"
	"github.com/albakov/go-cloud-file-storage/internal/logger"
	"github.com/albakov/go-cloud-file-storage/internal/storage"
	"github.com/go-sql-driver/mysql"
)

type Repository struct {
	pkg string
	db  *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		pkg: "usersession.repository",
		db:  db,
	}
}

func (us *Repository) ByRefreshToken(refreshToken string) (Session, error) {
	const op = "ByRefreshToken"

	var s Session
	err := us.db.QueryRow(
		"SELECT id, user_id, refresh_token, expires_at FROM users_sessions WHERE refresh_token = ?",
		refreshToken,
	).Scan(&s.Id, &s.UserId, &s.RefreshToken, &s.ExpiredAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, storage.ErrNotFound
		}

		return Session{}, logger.Error(us.pkg, op, err)
	}

	return s, nil
}

func (us *Repository) Create(userSession Session) (Session, error) {
	const op = "Create"

	stmt, err := us.db.Prepare("INSERT INTO users_sessions (user_id, refresh_token, expires_at) VALUES (?, ?, ?)")
	if err != nil {
		return Session{}, logger.Error(us.pkg, op, err)
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			logger.Add(us.pkg, op, err)
		}
	}(stmt)

	exec, err := stmt.Exec(userSession.UserId, userSession.RefreshToken, userSession.ExpiredAt)
	if err != nil {
		// check if error is because refresh_token duplicate
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return Session{}, storage.ErrDuplicateNotAllowed
		}

		return Session{}, logger.Error(us.pkg, op, err)
	}

	id, err := exec.LastInsertId()
	if err != nil {
		return Session{}, logger.Error(us.pkg, op, err)
	}

	userSession.Id = id

	return userSession, nil
}

func (us *Repository) Delete(userId int64, refreshToken string) error {
	const op = "Delete"

	stmt, err := us.db.Prepare("DELETE FROM users_sessions WHERE user_id = ? AND refresh_token = ?")
	if err != nil {
		return logger.Error(us.pkg, op, err)
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			logger.Add(us.pkg, op, err)
		}
	}(stmt)

	_, err = stmt.Exec(userId, refreshToken)
	if err != nil {
		return logger.Error(us.pkg, op, err)
	}

	return nil
}
