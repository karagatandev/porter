package porter_app

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/server/shared/requestutils"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/porter_app"
	"github.com/karagatandev/porter/internal/telemetry"
	"github.com/pkg/errors"
	porterv1 "github.com/porter-dev/api-contracts/generated/go/porter/v1"
)

// GetAppRevisionStatusHandler handles requests to the /apps/{porter_app_name}/revisions/{app_revision_id}/status endpoint
type GetAppRevisionStatusHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

// NewGetAppRevisionStatusHandler returns a new GetAppRevisionStatusHandler
func NewGetAppRevisionStatusHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *GetAppRevisionStatusHandler {
	return &GetAppRevisionStatusHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

// HighLevelStatus is a high level status that can be used to determine whether the revisions is progressing, successful or failed
type HighLevelStatus string

const (
	// HighLevelStatus_Progressing indicates that the revision is still in progress
	HighLevelStatus_Progressing HighLevelStatus = "progressing"
	// HighLevelStatus_Successful indicates that the revision has completed successfully
	HighLevelStatus_Successful HighLevelStatus = "successful"
	// HighLevelStatus_Failed indicates that the revision has failed
	HighLevelStatus_Failed HighLevelStatus = "failed"
)

// GetAppRevisionStatusResponse represents the response from the /apps/{porter_app_name}/revisions/{app_revision_id}/status endpoint
type GetAppRevisionStatusResponse struct {
	AppRevisionStatus porter_app.RevisionProgress `json:"app_revision_status"`
	// HighLevelStatus is a high level status that can be used to determine whether the revisions is progressing, successful or failed
	HighLevelStatus HighLevelStatus `json:"status"`
}

// GetAppRevisionStatusHandler returns the status of an app revision
func (c *GetAppRevisionStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.NewSpan(r.Context(), "serve-get-app-revision-status")
	defer span.End()

	project, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	appRevisionID, reqErr := requestutils.GetURLParamString(r, types.URLParamAppRevisionID)
	if reqErr != nil {
		err := telemetry.Error(ctx, span, nil, "error parsing app revision id")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusBadRequest))
		return
	}

	getRevisionStatusReq := connect.NewRequest(&porterv1.AppRevisionStatusRequest{
		ProjectId:     int64(project.ID),
		AppRevisionId: appRevisionID,
	})

	if c.Config().ClusterControlPlaneClient == nil {
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(errors.New("empty ClusterControlPlaneClient"), http.StatusInternalServerError))
		return
	}

	ccpResp, err := c.Config().ClusterControlPlaneClient.AppRevisionStatus(ctx, getRevisionStatusReq)
	if err != nil {
		err = telemetry.Error(ctx, span, err, "error getting app revision status")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	if ccpResp == nil || ccpResp.Msg == nil {
		err = telemetry.Error(ctx, span, nil, "get app revision response is nil")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	revisionStatus := porter_app.RevisionProgress{
		PredeployStarted:     ccpResp.Msg.PredeployStarted,
		PredeploySuccessful:  ccpResp.Msg.PredeploySuccessful,
		PredeployFailed:      ccpResp.Msg.PredeployFailed,
		InstallStarted:       ccpResp.Msg.InstallStarted,
		InstallSuccessful:    ccpResp.Msg.InstallSuccessful,
		InstallFailed:        ccpResp.Msg.InstallFailed,
		DeploymentStarted:    ccpResp.Msg.DeploymentStarted,
		DeploymentSuccessful: ccpResp.Msg.DeploymentSuccessful,
		DeploymentFailed:     ccpResp.Msg.DeploymentFailed,
		IsInTerminalStatus:   ccpResp.Msg.IsInTerminalStatus,
	}

	statusTransform := map[porterv1.EnumAppRevisionStatus]HighLevelStatus{
		porterv1.EnumAppRevisionStatus_ENUM_APP_REVISION_STATUS_PROGRESSING: HighLevelStatus_Progressing,
		porterv1.EnumAppRevisionStatus_ENUM_APP_REVISION_STATUS_SUCCESSFUL:  HighLevelStatus_Successful,
		porterv1.EnumAppRevisionStatus_ENUM_APP_REVISION_STATUS_FAILED:      HighLevelStatus_Failed,
	}

	status, ok := statusTransform[ccpResp.Msg.Status]
	if !ok {
		err = telemetry.Error(ctx, span, nil, "unsupported revision status status")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	res := &GetAppRevisionStatusResponse{
		AppRevisionStatus: revisionStatus,
		HighLevelStatus:   status,
	}

	c.WriteResult(w, r, res)
}
