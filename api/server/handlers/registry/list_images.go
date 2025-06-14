package registry

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/server/shared/requestutils"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/registry"
)

type RegistryListImagesHandler struct {
	handlers.PorterHandlerWriter
}

func NewRegistryListImagesHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *RegistryListImagesHandler {
	return &RegistryListImagesHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (c *RegistryListImagesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reg, _ := ctx.Value(types.RegistryScope).(*models.Registry)

	repoName, _ := requestutils.GetURLParamString(r, types.URLParamWildcard)

	// cast to a registry from registry package
	_reg := registry.Registry(*reg)
	regAPI := &_reg

	imgs, err := regAPI.ListImages(ctx, repoName, c.Repo(), c.Config())
	if err != nil {
		if strings.Contains(err.Error(), "RepositoryNotFoundException") {
			c.HandleAPIError(w, r, apierrors.NewErrNotFound(fmt.Errorf("no such repository: %s", repoName)))
			return
		}
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	c.WriteResult(w, r, imgs)
}
