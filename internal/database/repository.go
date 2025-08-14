package database

import (
	"context"

	"github.com/oxiginedev/go-session/internal/models"
)

type UserRepository interface {
	CreateUser(context.Context, *models.User) error
	FetchUserByEmail(context.Context, string) (*models.User, error)
}
