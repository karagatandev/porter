package job

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/server/shared/requestutils"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type StopHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

func NewStopHandler(
	config *config.Config,
) *StopHandler {
	return &StopHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, nil, nil),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *StopHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)
	agent, err := c.GetAgent(r, cluster, "")
	name, _ := requestutils.GetURLParamString(r, types.URLParamJobName)
	namespace, _ := requestutils.GetURLParamString(r, types.URLParamNamespace)

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	err = agent.StopJobWithJobSidecar(namespace, name)

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(
			err,
			http.StatusBadRequest,
		))
		return
	}
}
