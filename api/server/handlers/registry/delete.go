package registry

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type RegistryDeleteHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewRegistryDeleteHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *RegistryDeleteHandler {
	return &RegistryDeleteHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (p *RegistryDeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reg, _ := r.Context().Value(types.RegistryScope).(*models.Registry)

	if err := p.Repo().Registry().DeleteRegistry(reg); err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
	}

	return
}
