package jwt

import (
	"fmt"
	"github.com/albakov/go-cloud-file-storage/pkg/config"
	"github.com/albakov/go-cloud-file-storage/pkg/testutil"
	"path/filepath"
	"testing"
)

func TestJWT_GenerateAccessToken(t *testing.T) {
	dir, err := testutil.FindProjectRoot()
	if err != nil {
		t.Fatal(err)
	}

	jwtService := MustNew(config.MustNew(filepath.Join(dir, ".env.dev")))

	userId := int64(123456789)
	token, err := jwtService.GenerateAccessToken(userId)
	if err != nil {
		t.Error("generate access token error", err)
	}

	if token == "" {
		t.Error("empty access token")
	}
}

func TestJWT_ValidateAccessToken(t *testing.T) {
	dir, err := testutil.FindProjectRoot()
	if err != nil {
		t.Fatal(err)
	}

	jwtService := MustNew(config.MustNew(filepath.Join(dir, ".env.dev")))

	userId := int64(123456789)
	token, err := jwtService.GenerateAccessToken(userId)
	if err != nil {
		t.Error("generate access token error", err)
	}

	if token == "" {
		t.Error("empty access token")
	}

	accessToken, err := jwtService.ValidateAccessToken(token)
	if err != nil {
		t.Error("validate access token error", err)
	}

	if !accessToken.Valid {
		t.Error("invalid access token")
	}

	subject, err := accessToken.Claims.GetSubject()
	if err != nil {
		t.Error("get subject error", err)
	}

	userIdStr := fmt.Sprintf("%d", userId)

	if subject != userIdStr {
		t.Errorf("subject must be %v, got: %v", userIdStr, subject)
	}
}

func TestJWT_GenerateRefreshToken(t *testing.T) {
	dir, err := testutil.FindProjectRoot()
	if err != nil {
		t.Fatal(err)
	}

	jwtService := MustNew(config.MustNew(filepath.Join(dir, ".env.dev")))

	token, err := jwtService.GenerateRefreshToken()
	if err != nil {
		t.Error("generate refresh token error", err)
	}

	if token == "" {
		t.Error("empty refresh token")
	}
}
