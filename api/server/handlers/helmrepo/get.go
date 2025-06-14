package helmrepo

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type HelmRepoGetHandler struct {
	handlers.PorterHandlerWriter
}

func NewHelmRepoGetHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *HelmRepoGetHandler {
	return &HelmRepoGetHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (c *HelmRepoGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	helmRepo, _ := r.Context().Value(types.HelmRepoScope).(*models.HelmRepo)

	c.WriteResult(w, r, helmRepo.ToHelmRepoType())
}
