package cluster

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
)

type DeleteNamespaceHandler struct {
	handlers.PorterHandlerReader
	authz.KubernetesAgentGetter
}

func NewDeleteNamespaceHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
) *DeleteNamespaceHandler {
	return &DeleteNamespaceHandler{
		PorterHandlerReader:   handlers.NewDefaultPorterHandler(config, decoderValidator, nil),
		KubernetesAgentGetter: authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *DeleteNamespaceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)

	namespace, reqErr := requestutils.GetURLParamString(r, types.URLParamNamespace)

	if reqErr != nil {
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(reqErr, http.StatusBadRequest))
		return
	}

	agent, err := c.GetAgent(r, cluster, "")
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	if err := agent.DeleteNamespace(namespace); err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	w.WriteHeader(http.StatusOK)
}
