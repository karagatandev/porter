package cluster

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type ListClusterCandidatesHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewListClusterCandidatesHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *ListClusterCandidatesHandler {
	return &ListClusterCandidatesHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (c *ListClusterCandidatesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proj, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	ccs, err := c.Repo().Cluster().ListClusterCandidatesByProjectID(proj.ID)
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	res := make(types.ListClusterCandidateResponse, 0)

	for _, cc := range ccs {
		res = append(res, cc.ToClusterCandidateType())
	}

	c.WriteResult(w, r, res)
}
