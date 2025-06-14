//go:build ee
// +build ee

package usage

import (
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/ee/usage"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/repository"
)

var GetLimit func(repo repository.Repository, proj *models.Project) (limit *types.ProjectUsage, err error)

func init() {
	GetLimit = usage.GetLimit
}
