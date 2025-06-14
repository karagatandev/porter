package metadata

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
)

type ListRegistryIntegrationsHandler struct {
	handlers.PorterHandlerWriter
}

func NewListRegistryIntegrationsHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *ListRegistryIntegrationsHandler {
	return &ListRegistryIntegrationsHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (v *ListRegistryIntegrationsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v.WriteResult(w, r, types.PorterRegistryIntegrations)
}
