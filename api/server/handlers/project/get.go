package project

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type ProjectGetHandler struct {
	handlers.PorterHandlerWriter
}

func NewProjectGetHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *ProjectGetHandler {
	return &ProjectGetHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (p *ProjectGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proj, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	p.WriteResult(w, r, proj.ToProjectType(p.Config().LaunchDarklyClient))
}
