// NewGetUsageDashboardHandler returns a new GetUsageDashboardHandler
package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/telemetry"
)

// IngestEventsHandler is a handler for ingesting billing events
type IngestEventsHandler struct {
	handlers.PorterHandlerReadWriter
}

// NewIngestEventsHandler returns a new IngestEventsHandler
func NewIngestEventsHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *IngestEventsHandler {
	return &IngestEventsHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

func (c *IngestEventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.NewSpan(r.Context(), "serve-ingest-events")
	defer span.End()

	proj, _ := ctx.Value(types.ProjectScope).(*models.Project)

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "lago-config-exists", Value: c.Config().BillingManager.LagoConfigLoaded},
		telemetry.AttributeKV{Key: "lago-enabled", Value: proj.GetFeatureFlag(models.LagoEnabled, c.Config().LaunchDarklyClient)},
		telemetry.AttributeKV{Key: "porter-cloud-enabled", Value: proj.EnableSandbox},
	)

	if !c.Config().BillingManager.LagoConfigLoaded || !proj.GetFeatureFlag(models.LagoEnabled, c.Config().LaunchDarklyClient) {
		c.WriteResult(w, r, "")
		return
	}

	ingestEventsRequest := struct {
		Events []types.BillingEvent `json:"billing_events"`
	}{}

	if ok := c.DecodeAndValidate(w, r, &ingestEventsRequest); !ok {
		err := telemetry.Error(ctx, span, nil, "error decoding ingest events request")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusBadRequest))
		return
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "usage-events-count", Value: len(ingestEventsRequest.Events)},
	)

	var subscriptionID string
	if !proj.EnableSandbox {
		plan, err := c.Config().BillingManager.LagoClient.GetCustomerActivePlan(ctx, proj.ID, proj.EnableSandbox)
		if err != nil {
			err := telemetry.Error(ctx, span, err, "error getting active subscription")
			c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
			return
		}
		subscriptionID = plan.ID
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "subscription_id", Value: subscriptionID},
	)

	err := c.Config().BillingManager.LagoClient.IngestEvents(ctx, subscriptionID, ingestEventsRequest.Events, proj.EnableSandbox)
	if err != nil {
		err := telemetry.Error(ctx, span, err, "error ingesting events")
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	// Call the ingest health endpoint
	err = c.postIngestHealthEndpoint(ctx, proj.ID)
	if err != nil {
		err := telemetry.Error(ctx, span, err, "error calling ingest health endpoint")
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	c.WriteResult(w, r, "")
}

func (c *IngestEventsHandler) postIngestHealthEndpoint(ctx context.Context, projectID uint) (err error) {
	ctx, span := telemetry.NewSpan(ctx, "post-ingest-health-endpoint")
	defer span.End()

	// Call the ingest check webhook
	webhookUrl := c.Config().ServerConf.IngestStatusWebhookUrl
	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "ingest-status-webhook-url", Value: webhookUrl})

	if webhookUrl == "" {
		return nil
	}

	req := struct {
		ProjectID uint `json:"project_id"`
	}{
		ProjectID: projectID,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return telemetry.Error(ctx, span, err, "error marshalling ingest status webhook request")
	}

	client := &http.Client{}
	resp, err := client.Post(webhookUrl, "application/json", bytes.NewBuffer(reqBody))
	if err != nil || resp.StatusCode != http.StatusOK {
		return telemetry.Error(ctx, span, err, "error sending ingest status webhook request")
	}
	return nil
}
