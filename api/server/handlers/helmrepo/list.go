package helmrepo

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type HelmRepoListHandler struct {
	handlers.PorterHandlerWriter
}

func NewHelmRepoListHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *HelmRepoListHandler {
	return &HelmRepoListHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (c *HelmRepoListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proj, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	hrs, err := c.Repo().HelmRepo().ListHelmReposByProjectID(proj.ID)
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	res := make([]*types.HelmRepo, 0)

	for _, hr := range hrs {
		res = append(res, hr.ToHelmRepoType())
	}

	c.WriteResult(w, r, res)
}
