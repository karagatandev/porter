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

type ListIncidentsHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

func NewListIncidentsHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *ListIncidentsHandler {
	return &ListIncidentsHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *ListIncidentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)

	request := &types.ListIncidentsRequest{}

	if ok := c.DecodeAndValidate(w, r, request); !ok {
		return
	}

	agent, err := c.GetAgent(r, cluster, "")
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	// get agent service
	agentSvc, err := porter_agent.GetAgentService(agent.Clientset)
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	incidents, err := porter_agent.ListIncidents(agent.Clientset, agentSvc, request)
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	c.WriteResult(w, r, incidents)
}
