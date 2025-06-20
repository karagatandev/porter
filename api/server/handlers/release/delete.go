package release

import (
	"context"
	"net/http"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/integrations/ci/gitlab"
	"github.com/karagatandev/porter/internal/models"
	"github.com/stefanmcshane/helm/pkg/release"
)

type DeleteReleaseHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

func NewDeleteReleaseHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *DeleteReleaseHandler {
	return &DeleteReleaseHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *DeleteReleaseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(types.UserScope).(*models.User)
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)
	helmRelease, _ := r.Context().Value(types.ReleaseScope).(*release.Release)

	helmAgent, err := c.GetHelmAgent(r.Context(), r, cluster, "")
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	_, err = helmAgent.UninstallChart(context.Background(), helmRelease.Name)

	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	rel, releaseErr := c.Repo().Release().ReadRelease(cluster.ID, helmRelease.Name, helmRelease.Namespace)

	// update the github actions env if the release exists and is built from source
	if cName := helmRelease.Chart.Metadata.Name; cName == "job" || cName == "web" || cName == "worker" {
		if releaseErr == nil && rel != nil {
			gitAction := rel.GitActionConfig

			if gitAction != nil && gitAction.ID != 0 {
				if gitAction.GitlabIntegrationID != 0 {

					giRunner := &gitlab.GitlabCI{
						ServerURL:        c.Config().ServerConf.ServerURL,
						GitRepoPath:      gitAction.GitRepo,
						Repo:             c.Repo(),
						ProjectID:        cluster.ProjectID,
						ClusterID:        cluster.ID,
						UserID:           user.ID,
						IntegrationID:    gitAction.GitlabIntegrationID,
						PorterConf:       c.Config(),
						ReleaseName:      helmRelease.Name,
						ReleaseNamespace: helmRelease.Namespace,
					}

					err = giRunner.Cleanup()

					if err != nil {
						c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
						return
					}
				} else {
					gaRunner, err := GetGARunner(
						r.Context(),
						c.Config(),
						user.ID,
						cluster.ProjectID,
						cluster.ID,
						rel.GitActionConfig,
						helmRelease.Name,
						helmRelease.Namespace,
						rel,
						helmRelease,
					)
					if err != nil {
						c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
						return
					}

					err = gaRunner.Cleanup()

					if err != nil {
						c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
						return
					}
				}
			}
		}
	}
}
