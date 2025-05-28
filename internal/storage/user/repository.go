package user

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
		pkg: "user.repository",
		db:  db,
	}
}

func (u *Repository) Create(us User) (User, error) {
	const op = "Create"

	stmt, err := u.db.Prepare("INSERT INTO users (email, password) VALUES (?, ?)")
	if err != nil {
		return User{}, logger.Error(u.pkg, op, err)
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			logger.Add(u.pkg, op, err)
		}
	}(stmt)

	exec, err := stmt.Exec(us.Email, us.Password)
	if err != nil {
		// check if error is because email duplicate
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return User{}, storage.ErrDuplicateNotAllowed
		}

		return User{}, logger.Error(u.pkg, op, err)
	}

	id, err := exec.LastInsertId()
	if err != nil {
		return User{}, logger.Error(u.pkg, op, err)
	}

	us.Id = id

	return us, nil
}

func (u *Repository) IsExistsByEmail(email string) bool {
	const op = "IsExistsByEmail"

	var isExists int64
	err := u.db.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&isExists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false
		}

		logger.Add(u.pkg, op, err)

		return true
	}

	if isExists != 0 {
		return true
	}

	return false
}

func (u *Repository) ByEmail(email string) (User, error) {
	const op = "ByEmail"

	var us User
	err := u.db.QueryRow(
		"SELECT id, email, password FROM users WHERE email = ?",
		email,
	).Scan(&us.Id, &us.Email, &us.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, storage.ErrNotFound
		}

		return User{}, logger.Error(u.pkg, op, err)
	}

	return us, nil
}

func (u *Repository) ById(userId int64) (User, error) {
	const op = "ById"

	var us User
	err := u.db.QueryRow(
		"SELECT id, email, password FROM users WHERE id = ?",
		userId,
	).Scan(&us.Id, &us.Email, &us.Password)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, storage.ErrNotFound
		}

		return User{}, logger.Error(u.pkg, op, err)
	}

	return us, nil
}
