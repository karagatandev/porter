package release

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/server/shared/requestutils"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"gorm.io/gorm"
)

type UpdateBuildConfigHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewUpdateBuildConfigHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *UpdateBuildConfigHandler {
	return &UpdateBuildConfigHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

func (c *UpdateBuildConfigHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)
	name, _ := requestutils.GetURLParamString(r, types.URLParamReleaseName)
	namespace := r.Context().Value(types.NamespaceScope).(string)

	request := &types.UpdateBuildConfigRequest{}

	if ok := c.DecodeAndValidate(w, r, request); !ok {
		return
	}

	release, err := c.Repo().Release().ReadRelease(cluster.ID, name, namespace)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
	}

	config, err := json.Marshal(request.Config)
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	buildConfig := &models.BuildConfig{
		Builder:    request.Builder,
		Buildpacks: strings.Join(request.Buildpacks, ","),
		Config:     config,
	}

	buildConfig.ID = release.BuildConfig
	_, err = c.Repo().BuildConfig().UpdateBuildConfig(buildConfig)

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}
}
