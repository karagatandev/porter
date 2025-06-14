package env_group

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/server/shared/requestutils"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/kubernetes"
	"github.com/karagatandev/porter/internal/models"
)

type RemoveReleaseFromEnvGroupHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

func NewRemoveReleaseFromEnvGroupHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *RemoveReleaseFromEnvGroupHandler {
	return &RemoveReleaseFromEnvGroupHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *RemoveReleaseFromEnvGroupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	namespace := r.Context().Value(types.NamespaceScope).(string)
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)

	name, reqErr := requestutils.GetURLParamString(r, types.URLParamEnvGroupName)

	if reqErr != nil {
		c.HandleAPIError(w, r, reqErr)
		return
	}

	request := &types.V1EnvGroupReleaseRequest{}

	if ok := c.DecodeAndValidate(w, r, request); !ok {
		return
	}

	agent, err := c.GetAgent(r, cluster, "")
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	// read the attached configmap
	cm, _, err := agent.GetLatestVersionedConfigMap(name, namespace)

	if err != nil && errors.Is(err, kubernetes.IsNotFoundError) {
		c.HandleAPIError(w, r, apierrors.NewErrNotFound(fmt.Errorf("env group not found")))
		return
	} else if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	_, err = agent.RemoveApplicationFromVersionedConfigMap(cm, request.ReleaseName)

	if err != nil && errors.Is(err, kubernetes.IsNotFoundError) {
		c.HandleAPIError(w, r, apierrors.NewErrNotFound(fmt.Errorf("env group not found")))
		return
	} else if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}
}
