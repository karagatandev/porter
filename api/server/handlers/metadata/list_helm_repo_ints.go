package metadata

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
)

type ListHelmRepoIntegrationsHandler struct {
	handlers.PorterHandlerWriter
}

func NewListHelmRepoIntegrationsHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *ListHelmRepoIntegrationsHandler {
	return &ListHelmRepoIntegrationsHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (v *ListHelmRepoIntegrationsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v.WriteResult(w, r, types.PorterHelmRepoIntegrations)
}
