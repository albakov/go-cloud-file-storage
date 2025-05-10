package user_session_service

import (
	"database/sql"
	"errors"
	"github.com/albakov/go-cloud-file-storage/pkg/service/user_service"
	"github.com/albakov/go-cloud-file-storage/pkg/service/user_service/user_entity"
	"github.com/albakov/go-cloud-file-storage/pkg/service/user_session_service/user_session_entity"
	"github.com/albakov/go-cloud-file-storage/pkg/testutil"
	"testing"
	"time"
)

func TestUserSessionService_CreateUserSession(t *testing.T) {
	db, err := testutil.DbTest()
	if err != nil {
		t.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}(db)

	userService := user_service.New(db)

	u1, err := userService.CreateUser(user_entity.User{
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
	}(db, u1.Id)

	refreshToken := "1234"
	userSessionService := New(db)
	userSession, err := userSessionService.CreateUserSession(user_session_entity.UserSession{
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
			t.Errorf("error while delete user session: %v", err)
		}
	}(db, userSession.Id)

	u2, err := userService.UserByRefreshToken(refreshToken)
	if err != nil {
		t.Errorf("error while get user by refresh token: %v", err)
	}

	if u2.Id != u1.Id {
		t.Errorf("user by refresh token not match")
	}
}

func TestUserSessionService_CreateUserSessionDuplicate(t *testing.T) {
	db, err := testutil.DbTest()
	if err != nil {
		t.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}(db)

	userService := user_service.New(db)

	u1, err := userService.CreateUser(user_entity.User{
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
	}(db, u1.Id)

	refreshToken := "1234"
	userSessionService := New(db)

	userSession1, err := userSessionService.CreateUserSession(user_session_entity.UserSession{
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
			t.Errorf("error while delete user session: %v", err)
		}
	}(db, userSession1.Id)

	userSession2, err := userSessionService.CreateUserSession(user_session_entity.UserSession{
		UserId:       u1.Id,
		RefreshToken: refreshToken,
		ExpiredAt:    time.Now().Add(time.Hour * 24).Format(time.DateTime),
	})
	if err != nil {
		if !errors.Is(err, ErrAlreadyExists) {
			t.Errorf("error while create new user session: %v", err)
		}
	} else {
		deleteTestUserSession(db, userSession2.Id)
		t.Error("create duplicate user session not allowed")
	}
}

func TestUserSessionService_DeleteUserSession(t *testing.T) {
	db, err := testutil.DbTest()
	if err != nil {
		t.Fatal(err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			panic(err)
		}
	}(db)

	userService := user_service.New(db)

	u1, err := userService.CreateUser(user_entity.User{
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
	}(db, u1.Id)

	userSessionService := New(db)
	userSession, err := userSessionService.CreateUserSession(user_session_entity.UserSession{
		UserId:       u1.Id,
		RefreshToken: "1234",
		ExpiredAt:    time.Now().Add(time.Hour * 24).Format(time.DateTime),
	})
	if err != nil {
		t.Errorf("error while create new user session: %v", err)
	}
	defer func(db *sql.DB, userSessionId int64) {
		err := deleteTestUserSession(db, userSessionId)
		if err != nil {
			t.Errorf("error while delete user session: %v", err)
		}
	}(db, userSession.Id)

	err = userSessionService.DeleteUserSession(u1.Id, userSession.RefreshToken)
	if err != nil {
		t.Errorf("error while delete user session: %v", err)
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
