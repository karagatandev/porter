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

type ListOAuthHandler struct {
	handlers.PorterHandlerWriter
}

func NewListOAuthHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *ListOAuthHandler {
	return &ListOAuthHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (p *ListOAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	project, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	oauthInts, err := p.Repo().OAuthIntegration().ListOAuthIntegrationsByProjectID(project.ID)
	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	var res types.ListOAuthResponse = make([]*types.OAuthIntegration, 0)

	for _, oauthInt := range oauthInts {
		res = append(res, oauthInt.ToOAuthIntegrationType())
	}

	p.WriteResult(w, r, res)
}
