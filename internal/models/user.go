package models

import (
	"github.com/karagatandev/porter/api/types"
	"gorm.io/gorm"
)

// User type that extends gorm.Model
type User struct {
	gorm.Model

	Email         string `json:"email" gorm:"unique"`
	Password      string `json:"password"`
	EmailVerified bool   `json:"email_verified"`

	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	CompanyName string `json:"company_name"`

	// ID of oauth integration for github connection (optional)
	GithubAppIntegrationID uint

	// The github user id used for login (optional)
	GithubUserID int64
	GoogleUserID string

	AuthProvider string `json:"auth_provider"`
	ExternalId   string `json:"external_id"`
}

// AuthProvider_Ory represents the Ory auth provider
const AuthProvider_Ory = "ory"

// ToUserType generates an external types.User to be shared over REST
func (u *User) ToUserType() *types.User {
	return &types.User{
		ID:            u.ID,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		CompanyName:   u.CompanyName,
	}
}
