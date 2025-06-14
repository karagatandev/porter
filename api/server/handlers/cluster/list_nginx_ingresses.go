package cluster

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/kubernetes/prometheus"
	"github.com/karagatandev/porter/internal/models"
)

type ListNGINXIngressesHandler struct {
	handlers.PorterHandlerWriter
	authz.KubernetesAgentGetter
}

func NewListNGINXIngressesHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *ListNGINXIngressesHandler {
	return &ListNGINXIngressesHandler{
		PorterHandlerWriter:   handlers.NewDefaultPorterHandler(config, nil, writer),
		KubernetesAgentGetter: authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *ListNGINXIngressesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)

	agent, err := c.GetAgent(r, cluster, "")
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	ingresses, err := prometheus.GetIngressesWithNGINXAnnotation(agent.Clientset)
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	var res prometheus.ListNGINXIngressesResponse = ingresses

	c.WriteResult(w, r, res)
}
