package test

import (
	"context"

	ints "github.com/karagatandev/porter/internal/models/integrations"
	"github.com/karagatandev/porter/internal/repository"
)

type NeonIntegrationRepository struct{}

func NewNeonIntegrationRepository(canQuery bool) repository.NeonIntegrationRepository {
	return &NeonIntegrationRepository{}
}

func (s *NeonIntegrationRepository) Insert(ctx context.Context, neonInt ints.NeonIntegration) (ints.NeonIntegration, error) {
	panic("not implemented") // TODO: Implement
}

func (s *NeonIntegrationRepository) Integrations(ctx context.Context, projectID uint) ([]ints.NeonIntegration, error) {
	panic("not implemented") // TODO: Implement
}
