package repository

import (
	"github.com/karagatandev/porter/internal/models"
)

// AuthCodeRepository represents the set of queries on the AuthCode model
type AuthCodeRepository interface {
	CreateAuthCode(a *models.AuthCode) (*models.AuthCode, error)
	ReadAuthCode(code string) (*models.AuthCode, error)
}
