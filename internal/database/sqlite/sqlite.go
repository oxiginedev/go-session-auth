package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	inner *sql.DB
}

func New(ctx context.Context) (*DB, error) {
	db, err := sql.Open("sqlite3", "./internal/database/database.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database - %v", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return &DB{inner: db}, nil
}

func (s *DB) Close() error {
	return s.inner.Close()
}
