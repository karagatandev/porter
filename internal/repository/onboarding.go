package repository

import "github.com/karagatandev/porter/internal/models"

// ProjectOnboardingRepository represents the set of queries on the Onboarding model
type ProjectOnboardingRepository interface {
	CreateProjectOnboarding(onboarding *models.Onboarding) (*models.Onboarding, error)
	ReadProjectOnboarding(projID uint) (*models.Onboarding, error)
	UpdateProjectOnboarding(cache *models.Onboarding) (*models.Onboarding, error)
}
