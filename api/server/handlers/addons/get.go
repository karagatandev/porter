package addons

import (
	"encoding/base64"
	"net/http"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/server/shared/requestutils"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/telemetry"
	"github.com/porter-dev/api-contracts/generated/go/helpers"
	porterv1 "github.com/porter-dev/api-contracts/generated/go/porter/v1"
)

// AddonHandler handles requests to the /addons/{addon_name} endpoint
type AddonHandler struct {
	handlers.PorterHandlerReadWriter
}

// NewAddonHandler returns a new AddonHandler
func NewAddonHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *AddonHandler {
	return &AddonHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

// AddonResponse represents the response from the /addons/{addon_name} endpoints
type AddonResponse struct {
	Addon string `json:"addon"`
}

func (c *AddonHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.NewSpan(r.Context(), "serve-get-addon")
	defer span.End()

	project, _ := ctx.Value(types.ProjectScope).(*models.Project)
	deploymentTarget, _ := ctx.Value(types.DeploymentTargetScope).(types.DeploymentTarget)

	addonName, reqErr := requestutils.GetURLParamString(r, types.URLParamAddonName)
	if reqErr != nil {
		err := telemetry.Error(ctx, span, reqErr, "error parsing addon name")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusBadRequest))
		return
	}

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "addon-name", Value: addonName})

	var deploymentTargetIdentifier *porterv1.DeploymentTargetIdentifier
	if deploymentTarget.ID != uuid.Nil {
		deploymentTargetIdentifier = &porterv1.DeploymentTargetIdentifier{
			Id: deploymentTarget.ID.String(),
		}
	}

	if addonName == "" {
		err := telemetry.Error(ctx, span, nil, "no addon name provided")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusBadRequest))
		return
	}

	addonRequest := connect.NewRequest(&porterv1.AddonRequest{
		ProjectId:                  int64(project.ID),
		DeploymentTargetIdentifier: deploymentTargetIdentifier,
		AddonName:                  addonName,
	})

	resp, err := c.Config().ClusterControlPlaneClient.Addon(ctx, addonRequest)
	if err != nil {
		err = telemetry.Error(ctx, span, err, "error getting addon")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	if resp == nil || resp.Msg == nil {
		err = telemetry.Error(ctx, span, nil, "addon response is nil")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	by, err := helpers.MarshalContractObject(ctx, resp.Msg.Addon)
	if err != nil {
		err = telemetry.Error(ctx, span, err, "error marshaling addon")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	encoded := base64.StdEncoding.EncodeToString(by)

	res := &AddonResponse{
		Addon: encoded,
	}

	c.WriteResult(w, r, res)
}
