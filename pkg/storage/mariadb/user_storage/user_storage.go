package user_storage

import (
	"database/sql"
	"errors"
	"github.com/go-sql-driver/mysql"

	"github.com/albakov/go-cloud-file-storage/pkg/logger"
	"github.com/albakov/go-cloud-file-storage/pkg/storage"
	"github.com/albakov/go-cloud-file-storage/pkg/storage/entity/user"
)

type User struct {
	f  string
	db *sql.DB
}

func New(db *sql.DB) *User {
	return &User{
		f:  "user_storage",
		db: db,
	}
}

func (u *User) Create(us user.User) (user.User, error) {
	const op = "Create"

	stmt, err := u.db.Prepare("INSERT INTO users (email, password) VALUES (?, ?)")
	if err != nil {
		return user.User{}, logger.Error(u.f, op, err)
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			logger.Add(u.f, op, err)
		}
	}(stmt)

	exec, err := stmt.Exec(us.Email, us.Password)
	if err != nil {
		// check if error is because email duplicate
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return user.User{}, storage.ErrDuplicateNotAllowed
		}

		return user.User{}, logger.Error(u.f, op, err)
	}

	id, err := exec.LastInsertId()
	if err != nil {
		return user.User{}, logger.Error(u.f, op, err)
	}

	us.Id = id

	return us, nil
}

func (u *User) IsExistsByEmail(email string) bool {
	const op = "IsExistsByEmail"

	var isExists int64
	err := u.db.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&isExists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false
		}

		logger.Add(u.f, op, err)

		return true
	}

	if isExists != 0 {
		return true
	}

	return false
}

func (u *User) ByEmail(email string) (user.User, error) {
	const op = "ByEmail"

	var us user.User
	err := u.db.QueryRow(
		"SELECT id, email, password FROM users WHERE email = ?",
		email,
	).Scan(&us.Id, &us.Email, &us.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.User{}, storage.ErrNotFound
		}

		return user.User{}, logger.Error(u.f, op, err)
	}

	return us, nil
}

func (u *User) ById(userId int64) (user.User, error) {
	const op = "ById"

	var us user.User
	err := u.db.QueryRow(
		"SELECT id, email, password FROM users WHERE id = ?",
		userId,
	).Scan(&us.Id, &us.Email, &us.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.User{}, storage.ErrNotFound
		}

		return user.User{}, logger.Error(u.f, op, err)
	}

	return us, nil
}

func (u *User) ByRefreshToken(refreshToken string) (user.User, error) {
	const op = "ByRefreshToken"

	var us user.User
	err := u.db.QueryRow(
		`SELECT id, email, password FROM users 
        WHERE id = (SELECT user_id FROM users_sessions WHERE refresh_token = ? LIMIT 1)`,
		refreshToken,
	).Scan(&us.Id, &us.Email, &us.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user.User{}, storage.ErrNotFound
		}

		return user.User{}, logger.Error(u.f, op, err)
	}

	return us, nil
}
