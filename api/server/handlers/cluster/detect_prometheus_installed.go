package cluster

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/kubernetes/prometheus"
	"github.com/karagatandev/porter/internal/models"
)

type DetectPrometheusInstalledHandler struct {
	handlers.PorterHandler
	authz.KubernetesAgentGetter
}

func NewDetectPrometheusInstalledHandler(
	config *config.Config,
) *DetectPrometheusInstalledHandler {
	return &DetectPrometheusInstalledHandler{
		PorterHandler:         handlers.NewDefaultPorterHandler(config, nil, nil),
		KubernetesAgentGetter: authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *DetectPrometheusInstalledHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)

	agent, err := c.GetAgent(r, cluster, "")
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	if _, found, err := prometheus.GetPrometheusService(agent.Clientset); err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	} else if !found {
		http.NotFound(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
}
