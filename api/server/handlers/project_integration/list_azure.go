package project_integration

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type ListAzureHandler struct {
	handlers.PorterHandlerWriter
}

func NewListAzureHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *ListAzureHandler {
	return &ListAzureHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (p *ListAzureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	project, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	azInts, err := p.Repo().AzureIntegration().ListAzureIntegrationsByProjectID(project.ID)
	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	var res types.ListAzureResponse = make([]*types.AzureIntegration, 0)

	for _, azInt := range azInts {
		res = append(res, azInt.ToAzureIntegrationType())
	}

	p.WriteResult(w, r, res)
}
