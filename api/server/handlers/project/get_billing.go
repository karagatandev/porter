package project

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type ProjectGetBillingHandler struct {
	handlers.PorterHandlerWriter
}

func NewProjectGetBillingHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *ProjectGetBillingHandler {
	return &ProjectGetBillingHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (p *ProjectGetBillingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proj, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	res := &types.GetProjectBillingResponse{
		HasBilling: false,
	}

	if sc := p.Config().ServerConf; sc.BillingPrivateKey != "" && sc.BillingPrivateServerURL != "" {
		// determine if the project has usage attached; if so, set has_billing to true
		usage, _ := p.Repo().ProjectUsage().ReadProjectUsage(proj.ID)

		res.HasBilling = usage != nil
	}

	p.WriteResult(w, r, res)
}
