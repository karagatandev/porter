package project

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type ProjectListHandler struct {
	handlers.PorterHandlerWriter
}

func NewProjectListHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *ProjectListHandler {
	return &ProjectListHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (p *ProjectListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// read the user from context
	user, _ := r.Context().Value(types.UserScope).(*models.User)

	// read all projects for this user
	projects, err := p.Repo().Project().ListProjectsByUserID(user.ID)
	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	res := make([]*types.ProjectList, len(projects))

	for i, proj := range projects {
		res[i] = proj.ToProjectListType()
	}

	p.WriteResult(w, r, res)
}
