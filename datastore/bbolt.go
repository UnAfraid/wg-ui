package datastore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

func NewBBoltDB(databasePath string, timeout time.Duration) (*bbolt.DB, error) {
	dir := filepath.Dir(databasePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s - %w", dir, err)
	}

	db, err := bbolt.Open(databasePath, 0644, &bbolt.Options{
		Timeout: timeout,
	})
	if err != nil {
		return nil, err
	}

	if db.IsReadOnly() {
		return nil, errors.New("database is readonly")
	}

	return db, nil
}
