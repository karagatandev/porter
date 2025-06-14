package project

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
)

type RolesListHandler struct {
	handlers.PorterHandlerWriter
}

func NewRolesListHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *RolesListHandler {
	return &RolesListHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (p *RolesListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var res types.ListProjectRolesResponse = []types.RoleKind{types.RoleAdmin, types.RoleDeveloper, types.RoleViewer}

	p.WriteResult(w, r, res)
}
