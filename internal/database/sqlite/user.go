package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/oxiginedev/go-session/internal/database"
	"github.com/oxiginedev/go-session/internal/models"
)

type userRepo struct {
	db *DB
}

func NewUserRepository(db *DB) database.UserRepository {
	return &userRepo{
		db: db,
	}
}

func (u *userRepo) CreateUser(ctx context.Context, user *models.User) error {
	query := `
	INSERT INTO users(username, email, password)
	VALUES(?, ?, ?)
	`

	if _, err := u.db.inner.ExecContext(ctx, query,
		user.Username,
		user.Email,
		user.Password); err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return database.ErrUserExists
		}

		return err
	}

	return nil
}

func (u *userRepo) FetchUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User

	query := "SELECT * FROM users WHERE email = ? LIMIT 1"

	if err := u.db.inner.QueryRowContext(ctx, query, email).
		Scan(&user); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, database.ErrUserNotFound
		}

		return nil, err
	}

	return &user, nil
}
