package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/karagatandev/porter/internal/models"
)

// GithubWebhookRepository represents the set of queries on the GithubWebhook model
type GithubWebhookRepository interface {
	Insert(ctx context.Context, webhook *models.GithubWebhook) (*models.GithubWebhook, error)
	Get(ctx context.Context, id uuid.UUID) (*models.GithubWebhook, error)
	GetByClusterAndAppID(ctx context.Context, clusterID, appID uint) (*models.GithubWebhook, error)
}
