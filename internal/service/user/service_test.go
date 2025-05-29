package user

import (
	"database/sql"
	"errors"
	usersessionservice "github.com/albakov/go-cloud-file-storage/internal/service/usersession"
	"github.com/albakov/go-cloud-file-storage/internal/storage/user"
	"github.com/albakov/go-cloud-file-storage/internal/storage/usersession"
	"github.com/albakov/go-cloud-file-storage/internal/testutil"
	"testing"
	"time"
)

type testService struct {
	service *Service
	db      *sql.DB
}

func TestUserService_CreateUser(t *testing.T) {
	userService := userTestService(t)
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}(userService.db)

	userEntity := User{
		Email:    "test@example.ru",
		Password: "1234",
	}

	u1, err := userService.service.CreateUser(userEntity)
	if err != nil {
		t.Errorf("error while create new user: %v", err)
	}
	defer func(db *sql.DB, userId int64) {
		err := deleteTestUser(db, userId)
		if err != nil {
			t.Errorf("error while delete test user: %v", err)
		}
	}(userService.db, u1.Id)

	if u1.Email.String != userEntity.Email {
		t.Errorf("email does not match")
	}
}

func TestUserService_CreateUserDuplicate(t *testing.T) {
	userService := userTestService(t)
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}(userService.db)

	userEntity := User{
		Email:    "test@example.ru",
		Password: "1234",
	}

	u1, err := userService.service.CreateUser(userEntity)
	if err != nil {
		t.Errorf("error while create new user: %v", err)
	}
	defer func(db *sql.DB, userId int64) {
		err := deleteTestUser(db, userId)
		if err != nil {
			t.Errorf("error while delete test user: %v", err)
		}
	}(userService.db, u1.Id)

	// check when trying to create duplicate user
	u2, err := userService.service.CreateUser(userEntity)
	if err != nil {
		if !errors.Is(err, ErrAlreadyExists) {
			t.Errorf("error while create duplicate user: %v", err)
		}
	} else {
		deleteTestUser(userService.db, u2.Id)
		t.Error("create duplicate user not allowed")
	}
}

func TestUserService_UserByRefreshToken(t *testing.T) {
	userService := userTestService(t)
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}(userService.db)

	u1, err := userService.service.CreateUser(User{
		Email:    "test@example.ru",
		Password: "1234",
	})
	if err != nil {
		t.Errorf("error while create new user: %v", err)
	}
	defer func(db *sql.DB, userId int64) {
		err := deleteTestUser(db, userId)
		if err != nil {
			t.Errorf("error while delete test user: %v", err)
		}
	}(userService.db, u1.Id)

	refreshToken := "1234"

	userSessionRepo := usersession.NewRepository(userService.db)
	userSessionService := usersessionservice.NewService(userSessionRepo)

	userSession, err := userSessionService.CreateUserSession(usersessionservice.UserSession{
		UserId:       u1.Id,
		RefreshToken: refreshToken,
		ExpiredAt:    time.Now().Add(time.Hour * 24).Format(time.DateTime),
	})
	if err != nil {
		t.Errorf("error while create new user session: %v", err)
	}
	defer func(db *sql.DB, userSessionId int64) {
		err := deleteTestUserSession(db, userSessionId)
		if err != nil {
			t.Errorf("error while delete test user: %v", err)
		}
	}(userService.db, userSession.Id)

	u2, err := userSessionService.ValidUserSessionByRefreshToken(refreshToken)
	if err != nil {
		t.Errorf("error while get user by refresh token: %v", err)
	}

	if u2.UserId != u1.Id {
		t.Errorf("user by refresh token not match")
	}
}

func TestUserService_UserByEmail(t *testing.T) {
	userService := userTestService(t)
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}(userService.db)

	email := "test@example.ru"

	u1, err := userService.service.CreateUser(User{
		Email:    email,
		Password: "1234",
	})
	if err != nil {
		t.Errorf("error while create new user: %v", err)
	}
	defer func(db *sql.DB, userId int64) {
		err := deleteTestUser(db, userId)
		if err != nil {
			t.Errorf("error while delete test user: %v", err)
		}
	}(userService.db, u1.Id)

	u2, err := userService.service.UserByEmail(email)
	if err != nil {
		t.Errorf("error while get user by email: %v", err)
	}

	if u2.Id != u1.Id {
		t.Errorf("user by email not match")
	}
}

func deleteTestUser(db *sql.DB, userId int64) error {
	stmt, err := db.Prepare("DELETE FROM users WHERE id = ?")
	if err != nil {
		return err
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			return
		}
	}(stmt)
	_, err = stmt.Exec(userId)

	return err
}

func deleteTestUserSession(db *sql.DB, userSessionId int64) error {
	stmt, err := db.Prepare("DELETE FROM users_sessions WHERE id = ?")
	if err != nil {
		return err
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			return
		}
	}(stmt)
	_, err = stmt.Exec(userSessionId)

	return err
}

func userTestService(t *testing.T) *testService {
	db, err := testutil.DbTest()
	if err != nil {
		t.Fatal(err)
	}

	return &testService{
		service: NewService(user.NewRepository(db)),
		db:      db,
	}
}
