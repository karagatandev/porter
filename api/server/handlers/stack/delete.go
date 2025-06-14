package stack

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/kubernetes/envgroup"
	"github.com/karagatandev/porter/internal/models"
)

type StackDeleteHandler struct {
	handlers.PorterHandlerWriter
	authz.KubernetesAgentGetter
}

func NewStackDeleteHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *StackDeleteHandler {
	return &StackDeleteHandler{
		PorterHandlerWriter:   handlers.NewDefaultPorterHandler(config, nil, writer),
		KubernetesAgentGetter: authz.NewOutOfClusterAgentGetter(config),
	}
}

func (p *StackDeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	stack, _ := r.Context().Value(types.StackScope).(*models.Stack)
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)
	namespace, _ := r.Context().Value(types.NamespaceScope).(string)

	if len(stack.Revisions) > 0 {
		revision, err := p.Repo().Stack().ReadStackRevisionByNumber(stack.ID, stack.Revisions[0].RevisionNumber)
		if err != nil {
			p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
			return
		}

		k8sAgent, err := p.GetAgent(r, cluster, "")
		if err != nil {
			p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
			return
		}

		helmAgent, err := p.GetHelmAgent(r.Context(), r, cluster, namespace)
		if err != nil {
			p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
			return
		}

		// delete all resources in stack
		for _, appResource := range revision.Resources {
			deleteAppResource(&deleteAppResourceOpts{
				helmAgent: helmAgent,
				name:      appResource.Name,
			})
		}

		// delete all env groups in stack
		for _, envGroup := range revision.EnvGroups {
			envgroup.DeleteEnvGroup(k8sAgent, envGroup.Name, envGroup.Namespace)
		}
	}

	stack, err := p.Repo().Stack().DeleteStack(stack)
	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}
}
