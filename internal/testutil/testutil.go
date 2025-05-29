package testutil

import (
	"database/sql"
	"fmt"
	"github.com/albakov/go-cloud-file-storage/internal/config"
	"github.com/albakov/go-cloud-file-storage/internal/storage"
	"os"
	"path/filepath"
)

// FindProjectRoot returns project root path
func FindProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

// DbTest returns db resource for testing
func DbTest() (*sql.DB, error) {
	dir, err := FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("error while get dir path: %v", err)
	}

	return storage.MustNewClient(config.MustNew(filepath.Join(dir, ".env.dev")).MysqlDSN).DB(), nil
}
