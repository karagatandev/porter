package cluster

import (
	"net/http"

	"github.com/karagatandev/porter/internal/kubernetes/prometheus"
	"github.com/karagatandev/porter/internal/telemetry"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type GetPodMetricsHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

func NewGetPodMetricsHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *GetPodMetricsHandler {
	return &GetPodMetricsHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *GetPodMetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := telemetry.NewSpan(ctx, "service-get-pod-metrics")
	defer span.End()

	cluster, _ := ctx.Value(types.ClusterScope).(*models.Cluster)

	request := &prometheus.GetPodMetricsRequest{}

	if ok := c.DecodeAndValidate(w, r, request); !ok {
		err := telemetry.Error(ctx, span, nil, "error decoding request")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusBadRequest))
		return
	}

	agent, err := c.GetAgent(r, cluster, "")
	if err != nil {
		err = telemetry.Error(ctx, span, err, "error getting k8s agent")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	// get prometheus service
	promSvc, found, err := prometheus.GetPrometheusService(agent.Clientset)
	if err != nil || !found {
		err = telemetry.Error(ctx, span, err, "error getting prometheus service")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	rawQuery, err := prometheus.QueryPrometheus(ctx, agent.Clientset, promSvc, &request.QueryOpts)
	if err != nil {
		err = telemetry.Error(ctx, span, err, "error querying prometheus")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	c.WriteResult(w, r, rawQuery)
}
