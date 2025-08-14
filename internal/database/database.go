package database

import (
	"database/sql"
	"errors"
)

var (
	ErrUserExists   = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
)

type Database interface {
	// GetDB gets the underlying database connection
	GetDB() *sql.DB
	// Close closes the database connection
	Close() error
}
