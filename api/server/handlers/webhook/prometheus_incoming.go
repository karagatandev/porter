package webhook

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/server/shared/requestutils"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/telemetry"
	porterv1 "github.com/porter-dev/api-contracts/generated/go/porter/v1"
)

// PrometheusAlertWebhookHandler handles incoming prometheus alerts
type PrometheusAlertWebhookHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

// NewPrometheusAlertWebhookHandler returns an instance of PrometheusAlertWebhookHandler
func NewPrometheusAlertWebhookHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *PrometheusAlertWebhookHandler {
	return &PrometheusAlertWebhookHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

func (p *PrometheusAlertWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.NewSpan(r.Context(), "serve-post-prometheus-alert")
	defer span.End()

	// get the webhook id from the request
	projectID, err := requestutils.GetURLParamUint(r, types.URLParamProjectID)
	if err != nil {
		e := telemetry.Error(ctx, span, err, "error getting project ID")
		p.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(e, http.StatusBadRequest))
		return
	}
	clusterID, err := requestutils.GetURLParamUint(r, types.URLParamClusterID)
	if err != nil {
		e := telemetry.Error(ctx, span, nil, "error getting cluster ID")
		p.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(e, http.StatusBadRequest))
		return
	}

	prometheusAlert := &types.PrometheusAlert{}
	if ok := p.DecodeAndValidate(w, r, prometheusAlert); !ok {
		e := telemetry.Error(ctx, span, nil, "error decoding request")
		p.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(e, http.StatusBadRequest))
		return
	}
	if err := p.handlePrometheusAlert(ctx, int64(projectID), int64(clusterID), prometheusAlert); err != nil {
		e := telemetry.Error(ctx, span, err, "error handling prometheus alert")
		p.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(e, http.StatusInternalServerError))
		return
	}
	p.WriteResult(w, r, "")
}

func (p *PrometheusAlertWebhookHandler) handlePrometheusAlert(ctx context.Context, projectId, clusterId int64, prometheusAlert *types.PrometheusAlert) error {
	ctx, span := telemetry.NewSpan(ctx, "porter-process-prom-alert")
	defer span.End()
	recordPrometheusAlertRequest := connect.NewRequest(&porterv1.RecordPrometheusAlertRequest{
		ProjectId: projectId,
		ClusterId: clusterId,
	})
	labelKeyValues := ""
	for _, alert := range prometheusAlert.Alerts {
		for k, v := range alert.Labels {
			labelKeyValues += fmt.Sprintf("%s %s", k, v)
		}
		if alert.Labels["alertname"] == "NoopAlert" {
			continue
		}
		startTime, err := time.Parse(time.RFC3339, alert.StartsAt)
		if err != nil {
			return telemetry.Error(ctx, span, err, "error parsing alert start time")
		}
		endTime, err := time.Parse(time.RFC3339, alert.EndsAt)
		if err != nil {
			return telemetry.Error(ctx, span, err, "error parsing alert end time")
		}
		var endTimestamp *timestamppb.Timestamp
		if endTime.After(startTime) {
			endTimestamp = timestamppb.New(endTime)
		}
		recordPrometheusAlertRequest.Msg.Alerts = append(recordPrometheusAlertRequest.Msg.Alerts, &porterv1.Alert{
			Name:      alert.Labels["name"],
			Namespace: alert.Labels["namespace"],
			Type:      p.getType(alert),
			Severity:  alert.Labels["severity"],
			StartTime: timestamppb.New(startTime),
			EndTime:   endTimestamp,
		})
	}
	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "porter-app-alert-labels", Value: labelKeyValues})
	_, err := p.Config().ClusterControlPlaneClient.RecordPrometheusAlert(ctx, recordPrometheusAlertRequest)
	if err != nil {
		return telemetry.Error(ctx, span, err, "error recording prometheus alert")
	}
	return nil
}

func (p *PrometheusAlertWebhookHandler) getType(alert types.Alert) porterv1.InvolvedObjectType {
	switch alert.Labels["involvedObjectType"] {
	case "Deployment":
		return porterv1.InvolvedObjectType_INVOLVED_OBJECT_TYPE_DEPLOYMENT
	case "StatefulSet":
		return porterv1.InvolvedObjectType_INVOLVED_OBJECT_TYPE_STATEFULSET
	case "DaemonSet":
		return porterv1.InvolvedObjectType_INVOLVED_OBJECT_TYPE_DAEMONSET
	default:
		return porterv1.InvolvedObjectType_INVOLVED_OBJECT_TYPE_UNSPECIFIED
	}
}
