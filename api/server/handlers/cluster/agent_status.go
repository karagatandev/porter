package cluster

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	porter_agent "github.com/karagatandev/porter/internal/kubernetes/porter_agent/v2"
	"github.com/karagatandev/porter/internal/models"
)

type GetAgentStatusHandler struct {
	handlers.PorterHandlerWriter
	authz.KubernetesAgentGetter
}

func NewGetAgentStatusHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *GetAgentStatusHandler {
	return &GetAgentStatusHandler{
		PorterHandlerWriter:   handlers.NewDefaultPorterHandler(config, nil, writer),
		KubernetesAgentGetter: authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *GetAgentStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)

	k8sAgent, err := c.GetAgent(r, cluster, "porter-agent-system")
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	// get agent service
	agentSvc, err := porter_agent.GetAgentService(k8sAgent.Clientset)
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	status, err := porter_agent.GetAgentStatus(k8sAgent.Clientset, agentSvc)
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	c.WriteResult(w, r, status)
}
