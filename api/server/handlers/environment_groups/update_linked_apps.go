package environment_groups

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/telemetry"
	porterv1 "github.com/porter-dev/api-contracts/generated/go/porter/v1"
)

// UpdateLinkedAppsHandler is the handle for the /environment-group/update-linked-apps endpoint
type UpdateLinkedAppsHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

// NewUpdateLinkedAppsHandler creates an instance of UpdateLinkedAppsHandler
func NewUpdateLinkedAppsHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *UpdateLinkedAppsHandler {
	return &UpdateLinkedAppsHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

// UpdateLinkedAppsRequest is the request object for the /environment-group/update-linked-apps endpoint
type UpdateLinkedAppsRequest struct {
	Name string `json:"name"`
}

// UpdateLinkedAppsResponse is the response object for the /environment-group/update-linked-apps endpoint
type UpdateLinkedAppsResponse struct{}

// ServeHTTP updates all apps linked to an environment group
func (c *UpdateLinkedAppsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.NewSpan(r.Context(), "serve-update-apps-linked-to-env-group")
	defer span.End()

	project, _ := ctx.Value(types.ProjectScope).(*models.Project)
	cluster, _ := ctx.Value(types.ClusterScope).(*models.Cluster)

	if !project.GetFeatureFlag(models.ValidateApplyV2, c.Config().LaunchDarklyClient) {
		err := telemetry.Error(ctx, span, nil, "project does not have validate apply v2 enabled")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusForbidden))
		return
	}

	request := &UpdateLinkedAppsRequest{}
	if ok := c.DecodeAndValidate(w, r, request); !ok {
		err := telemetry.Error(ctx, span, nil, "error decoding request")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusBadRequest))
		return
	}
	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "env-group-name", Value: request.Name})

	updateLinkedAppsReq := connect.NewRequest(&porterv1.UpdateAppsLinkedToEnvGroupRequest{
		ProjectId:    int64(project.ID),
		ClusterId:    int64(cluster.ID),
		EnvGroupName: request.Name,
	})
	_, err := c.Config().ClusterControlPlaneClient.UpdateAppsLinkedToEnvGroup(ctx, updateLinkedAppsReq)
	if err != nil {
		err := telemetry.Error(ctx, span, err, "error calling ccp update apps linked to env group")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	res := &UpdateLinkedAppsResponse{}

	c.WriteResult(w, r, res)
}
