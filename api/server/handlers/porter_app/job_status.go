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

// JobStatusHandler is the handler for GET /apps/jobs
type JobStatusHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

// NewJobStatusHandler returns a new JobStatusHandler
func NewJobStatusHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *JobStatusHandler {
	return &JobStatusHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

// JobStatusRequest is the expected format for a request body on GET /apps/jobs
type JobStatusRequest struct {
	DeploymentTargetID   string `schema:"deployment_target_id,omitempty"`
	DeploymentTargetName string `schema:"deployment_target_name,omitempty"`
	JobName              string `schema:"job_name"`
}

// JobStatusResponse is the response format for GET /apps/jobs
type JobStatusResponse struct {
	JobRuns []porter_app.JobRun `json:"job_runs"`
}

func (c *JobStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.NewSpan(r.Context(), "serve-job-status")
	defer span.End()

	request := &JobStatusRequest{}
	if ok := c.DecodeAndValidate(w, r, request); !ok {
		err := telemetry.Error(ctx, span, nil, "invalid request")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusBadRequest))
		return
	}

	cluster, _ := ctx.Value(types.ClusterScope).(*models.Cluster)
	project, _ := ctx.Value(types.ProjectScope).(*models.Project)

	name, reqErr := requestutils.GetURLParamString(r, types.URLParamPorterAppName)
	if reqErr != nil {
		err := telemetry.Error(ctx, span, reqErr, "invalid porter app name")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusBadRequest))
		return
	}

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "app-name", Value: name})

	if c.Config().ClusterControlPlaneClient == nil {
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(errors.New("empty ClusterControlPlaneClient"), http.StatusInternalServerError))
		return
	}

	deploymentTargetName := request.DeploymentTargetName
	if request.DeploymentTargetName == "" && request.DeploymentTargetID == "" {
		defaultDeploymentTarget, err := defaultDeploymentTarget(ctx, defaultDeploymentTargetInput{
			ProjectID:                 project.ID,
			ClusterID:                 cluster.ID,
			ClusterControlPlaneClient: c.Config().ClusterControlPlaneClient,
		})
		if err != nil {
			err := telemetry.Error(ctx, span, err, "error getting default deployment target")
			c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
			return
		}
		deploymentTargetName = defaultDeploymentTarget.Name
	}

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "deployment-target-name", Value: deploymentTargetName},
		telemetry.AttributeKV{Key: "deployment-target-id", Value: request.DeploymentTargetID},
	)

	jobRunsRequest := connect.NewRequest(&porterv1.JobRunsRequest{
		ProjectId: int64(project.ID),
		DeploymentTargetIdentifier: &porterv1.DeploymentTargetIdentifier{
			Id:   request.DeploymentTargetID,
			Name: deploymentTargetName,
		},
		AppName:        name,
		JobServiceName: request.JobName,
	})

	if c.Config().ClusterControlPlaneClient == nil {
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(errors.New("empty ClusterControlPlaneClient"), http.StatusInternalServerError))
		return
	}

	jobRunsResp, err := c.Config().ClusterControlPlaneClient.JobRuns(ctx, jobRunsRequest)
	if err != nil {
		err := telemetry.Error(ctx, span, err, "error getting job runs from cluster control plane client")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusBadRequest))
		return
	}

	if jobRunsResp == nil || jobRunsResp.Msg == nil {
		err := telemetry.Error(ctx, span, nil, "job runs response is nil")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	runs := []porter_app.JobRun{}
	for _, jobRun := range jobRunsResp.Msg.JobRuns {
		run, err := porter_app.JobRunFromProto(ctx, jobRun)
		if err != nil {
			err := telemetry.Error(ctx, span, err, "error converting job run from proto")
			c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
			return
		}

		runs = append(runs, run)
	}

	res := JobStatusResponse{
		JobRuns: runs,
	}

	c.WriteResult(w, r, res)
}
