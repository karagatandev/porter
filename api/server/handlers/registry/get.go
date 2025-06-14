package registry

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type RegistryGetHandler struct {
	handlers.PorterHandlerWriter
}

func NewRegistryGetHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *RegistryGetHandler {
	return &RegistryGetHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (c *RegistryGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	registry, _ := r.Context().Value(types.RegistryScope).(*models.Registry)

	c.WriteResult(w, r, registry.ToRegistryType())
}
