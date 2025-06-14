package release

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/server/shared/requestutils"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"gorm.io/gorm"
)

type GetReleaseStepsHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

func NewGetReleaseStepsHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *GetReleaseStepsHandler {
	return &GetReleaseStepsHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *GetReleaseStepsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)
	name, _ := requestutils.GetURLParamString(r, types.URLParamReleaseName)
	namespace := r.Context().Value(types.NamespaceScope).(string)

	release, err := c.Repo().Release().ReadRelease(cluster.ID, name, namespace)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	res := make(types.GetReleaseStepsResponse, 0)

	if release.EventContainer != 0 {
		subevents, err := c.Repo().BuildEvent().ReadEventsByContainerID(release.EventContainer)
		if err != nil {
			c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
			return
		}

		for _, sub := range subevents {
			res = append(res, sub.ToSubEventType())
		}
	}

	c.WriteResult(w, r, res)
}
