package namespace

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
	"github.com/karagatandev/porter/api/server/shared/websocket"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/kubernetes"
	"github.com/karagatandev/porter/internal/models"
)

type StreamPodLogsHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

func NewStreamPodLogsHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *StreamPodLogsHandler {
	return &StreamPodLogsHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *StreamPodLogsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	request := &types.GetPodLogsRequest{}

	if ok := c.DecodeAndValidate(w, r, request); !ok {
		return
	}

	safeRW := r.Context().Value(types.RequestCtxWebsocketKey).(*websocket.WebsocketSafeReadWriter)
	namespace := r.Context().Value(types.NamespaceScope).(string)
	name, _ := requestutils.GetURLParamString(r, types.URLParamPodName)

	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)

	agent, err := c.GetAgent(r, cluster, "")
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	err = agent.GetPodLogs(namespace, name, request.Container, safeRW)

	if err != nil {
		if errors.Is(err, kubernetes.IsNotFoundError) {
			c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(fmt.Errorf("pod %s/%s was not found", namespace, name),
				http.StatusNotFound))
			return
		} else if _, ok := err.(*kubernetes.BadRequestError); ok {
			c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusBadRequest))
			return
		} else {
			c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
			return
		}
	}
}
