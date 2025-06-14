//go:build !ee
// +build !ee

package usage

import (
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/repository"
)

func GetLimit(repo repository.Repository, proj *models.Project) (limit *types.ProjectUsage, err error) {
	copyLimit := types.BasicPlan

	return &copyLimit, nil
}
