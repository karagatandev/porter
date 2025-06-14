package porter_app

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

// AttachEnvGroupHandler is the handler for the /apps/attach-env-group endpoint
type AttachEnvGroupHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

// NewAttachEnvGroupHandler handles POST requests to the endpoint /apps/attach-env-group
func NewAttachEnvGroupHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *AttachEnvGroupHandler {
	return &AttachEnvGroupHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

// AttachEnvGroupRequest is the request object for the /apps/attach-env-group endpoint
type AttachEnvGroupRequest struct {
	EnvGroupName   string   `json:"env_group_name"`
	AppInstanceIDs []string `json:"app_instance_ids"`
}

// ServeHTTP translates the request into a AttachEnvGroup request, then calls update on the app with the env group
// The latest version of the env group will be attached (ccp makes sure of that)
func (c *AttachEnvGroupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.NewSpan(r.Context(), "serve-attach-env-group")
	defer span.End()

	project, _ := ctx.Value(types.ProjectScope).(*models.Project)

	request := &AttachEnvGroupRequest{}
	if ok := c.DecodeAndValidate(w, r, request); !ok {
		err := telemetry.Error(ctx, span, nil, "error decoding request")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusBadRequest))
		return
	}

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "env-group-name", Value: request.EnvGroupName})

	if request.EnvGroupName == "" {
		err := telemetry.Error(ctx, span, nil, "env group name cannot be empty")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusBadRequest))
		return
	}

	for _, appInstanceId := range request.AppInstanceIDs {
		appInstance, err := c.Repo().AppInstance().Get(ctx, appInstanceId)
		if err != nil {
			telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "app-instance-id", Value: appInstanceId})
			err := telemetry.Error(ctx, span, err, "error getting app instance")
			c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
			return
		}

		updateReq := connect.NewRequest(&porterv1.UpdateAppRequest{
			ProjectId: int64(project.ID),
			DeploymentTargetIdentifier: &porterv1.DeploymentTargetIdentifier{
				Id: appInstance.DeploymentTargetID.String(),
			},
			App: &porterv1.PorterApp{
				Name: appInstance.Name,
				EnvGroups: []*porterv1.EnvGroup{
					{
						Name: request.EnvGroupName,
					},
				},
			},
		})

		_, err = c.Config().ClusterControlPlaneClient.UpdateApp(ctx, updateReq)
		if err != nil {
			telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "app-instance-id", Value: appInstanceId})
			err := telemetry.Error(ctx, span, err, "error calling ccp update app")
			c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
			return
		}

	}

	c.WriteResult(w, r, nil)
}
