package project

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type GetTagsHandler struct {
	handlers.PorterHandlerWriter
}

func NewGetTagsHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *GetTagsHandler {
	return &GetTagsHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (p *GetTagsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proj, _ := r.Context().Value(types.ProjectScope).(*models.Project)
	tags, err := p.Repo().Tag().ListTagsByProjectId(proj.ID)
	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
	}

	p.WriteResult(w, r, tags)
}
