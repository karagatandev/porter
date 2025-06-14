package cluster

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
	"github.com/karagatandev/porter/internal/repository"
	"github.com/karagatandev/porter/internal/telemetry"
	porterv1 "github.com/porter-dev/api-contracts/generated/go/porter/v1"
)

type ClusterDeleteHandler struct {
	handlers.PorterHandlerWriter
	authz.KubernetesAgentGetter
}

func NewClusterDeleteHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *ClusterDeleteHandler {
	return &ClusterDeleteHandler{
		PorterHandlerWriter:   handlers.NewDefaultPorterHandler(config, nil, writer),
		KubernetesAgentGetter: authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *ClusterDeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.NewSpan(r.Context(), "serve-delete-cluster")
	defer span.End()

	cluster, _ := ctx.Value(types.ClusterScope).(*models.Cluster)

	if cluster.ProvisionedBy == "CAPI" {
		if c.Config().EnableCAPIProvisioner {
			revisions, err := c.Config().Repo.APIContractRevisioner().List(ctx, cluster.ProjectID, repository.WithClusterID(cluster.ID))
			if err != nil {
				err = telemetry.Error(ctx, span, err, "error listing revisions for cluster")
				c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
				return
			}
			if cluster.Status == types.UpdatingUnavailable || cluster.Status == types.Updating {
				err = telemetry.Error(ctx, span, nil, "unable to delete cluster that is updating")
				c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
				return
			}
			var revisionID string
			for _, rev := range revisions {
				if rev.Condition != "" {
					revisionID = rev.ID.String()
					break
				}
			}
			cl := connect.NewRequest(&porterv1.DeleteClusterRequest{
				ContractRevision: &porterv1.ContractRevision{
					ClusterId:  int32(cluster.ID),
					ProjectId:  int32(cluster.ProjectID),
					RevisionId: revisionID,
				},
			})
			_, err = c.Config().ClusterControlPlaneClient.DeleteCluster(ctx, cl)
			if err != nil {
				err = telemetry.Error(ctx, span, err, "error deleting cluster")
				c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
				return
			}
		}
	}

	err := c.Repo().Cluster().DeleteCluster(cluster)
	if err != nil {
		err = telemetry.Error(ctx, span, err, "error deleting cluster")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	c.WriteResult(w, r, cluster.ToClusterType())
}
