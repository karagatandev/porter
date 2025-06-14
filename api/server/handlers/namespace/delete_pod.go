package namespace

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/server/shared/requestutils"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/kubernetes"
	"github.com/karagatandev/porter/internal/models"
)

type DeletePodHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

func NewDeletePodHandler(
	config *config.Config,
) *DeletePodHandler {
	return &DeletePodHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, nil, nil),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *DeletePodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)
	agent, err := c.GetAgent(r, cluster, "")
	name, _ := requestutils.GetURLParamString(r, types.URLParamPodName)
	namespace, _ := requestutils.GetURLParamString(r, types.URLParamNamespace)

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	err = agent.DeletePod(namespace, name)

	if targetErr := kubernetes.IsNotFoundError; errors.Is(err, targetErr) {
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(
			fmt.Errorf("pod %s/%s was not found", namespace, name),
			http.StatusNotFound,
		))

		return
	} else if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}
}
