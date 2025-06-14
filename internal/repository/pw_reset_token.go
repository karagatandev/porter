package repository

import (
	"github.com/karagatandev/porter/internal/models"
)

// PWResetTokenRepository represents the set of queries on the PWResetToken model
type PWResetTokenRepository interface {
	CreatePWResetToken(pwToken *models.PWResetToken) (*models.PWResetToken, error)
	ReadPWResetToken(id uint) (*models.PWResetToken, error)
	UpdatePWResetToken(pwToken *models.PWResetToken) (*models.PWResetToken, error)
}
