package cluster

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/server/shared/requestutils"
	"github.com/karagatandev/porter/api/server/shared/websocket"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type StreamStatusHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

func NewStreamStatusHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *StreamStatusHandler {
	return &StreamStatusHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *StreamStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	safeRW := r.Context().Value(types.RequestCtxWebsocketKey).(*websocket.WebsocketSafeReadWriter)
	request := &types.StreamStatusRequest{}

	if ok := c.DecodeAndValidate(w, r, request); !ok {
		return
	}

	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)

	agent, err := c.GetAgent(r, cluster, "")
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	kind, _ := requestutils.GetURLParamString(r, types.URLParamKind)

	err = agent.StreamControllerStatus(kind, request.Selectors, safeRW)

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}
}
