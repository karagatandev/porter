package release

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"github.com/stefanmcshane/helm/pkg/release"
)

type CreateWebhookHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewCreateWebhookHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *CreateWebhookHandler {
	return &CreateWebhookHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (c *CreateWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	helmRelease, _ := r.Context().Value(types.ReleaseScope).(*release.Release)
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)

	release, err := CreateAppReleaseFromHelmRelease(r.Context(), c.Config(), cluster.ProjectID, cluster.ID, 0, helmRelease)
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	c.WriteResult(w, r, release.ToReleaseType())
}
