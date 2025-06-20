package release

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	semver "github.com/Masterminds/semver/v3"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	baseReleaseHandler "github.com/karagatandev/porter/api/server/handlers/release"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/helm"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/notifier"
	"github.com/karagatandev/porter/internal/notifier/slack"
	"github.com/stefanmcshane/helm/pkg/release"
)

var createEnvSecretConstraint, _ = semver.NewConstraint(" < 0.1.0")

type UpgradeReleaseHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

func NewUpgradeReleaseHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *UpgradeReleaseHandler {
	return &UpgradeReleaseHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *UpgradeReleaseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(types.UserScope).(*models.User)
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)
	helmRelease, _ := r.Context().Value(types.ReleaseScope).(*release.Release)

	helmAgent, err := c.GetHelmAgent(r.Context(), r, cluster, "")
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	request := &types.V1UpgradeReleaseRequest{}

	if ok := c.DecodeAndValidate(w, r, request); !ok {
		return
	}

	registries, err := c.Repo().Registry().ListRegistriesByProjectID(cluster.ProjectID)
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	conf := &helm.UpgradeReleaseConfig{
		Name:       helmRelease.Name,
		Cluster:    cluster,
		Repo:       c.Repo(),
		Registries: registries,
	}

	// if the chart version is set, load a chart from the repo
	if request.ChartVersion != "" {
		cache := c.Config().URLCache
		chartRepoURL, foundFirst := cache.GetURL(helmRelease.Chart.Metadata.Name)

		if !foundFirst {
			cache.Update()

			var found bool

			chartRepoURL, found = cache.GetURL(helmRelease.Chart.Metadata.Name)

			if !found {
				c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(
					fmt.Errorf("chart not found"),
					http.StatusBadRequest,
				))

				return
			}
		}

		chart, err := baseReleaseHandler.LoadChart(r.Context(), c.Config(), &baseReleaseHandler.LoadAddonChartOpts{
			ProjectID:       cluster.ProjectID,
			RepoURL:         chartRepoURL,
			TemplateName:    helmRelease.Chart.Metadata.Name,
			TemplateVersion: request.ChartVersion,
		})
		if err != nil {
			c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
			return
		}

		if err != nil {
			c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(
				fmt.Errorf("chart not found"),
				http.StatusBadRequest,
			))

			return
		}

		conf.Chart = chart
	}

	conf.Values = request.Values

	// if LatestRevision is set, check that the revision matches the latest revision in the database
	if request.LatestRevision != 0 {
		currHelmRelease, err := helmAgent.GetRelease(context.Background(), helmRelease.Name, 0, false)
		if err != nil {
			c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(
				fmt.Errorf("could not retrieve latest revision"),
				http.StatusBadRequest,
			))

			return
		}

		if currHelmRelease.Version != int(request.LatestRevision) {
			c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(
				fmt.Errorf("The provided revision is not up to date with the current revision (you may need to refresh the deployment). Provided revision is %d, latest revision is %d. If you would like to deploy from this revision, please revert first and update the configuration.", request.LatestRevision, currHelmRelease.Version),
				http.StatusBadRequest,
			))

			return
		}
	}

	newHelmRelease, upgradeErr := helmAgent.UpgradeReleaseByValues(context.Background(), conf, c.Config().DOConf, c.Config().ServerConf.DisablePullSecretsInjection, false)

	if upgradeErr == nil && newHelmRelease != nil {
		helmRelease = newHelmRelease
	}

	slackInts, _ := c.Repo().SlackIntegration().ListSlackIntegrationsByProjectID(cluster.ProjectID)

	rel, releaseErr := c.Repo().Release().ReadRelease(cluster.ID, helmRelease.Name, helmRelease.Namespace)

	var notifConf *types.NotificationConfig
	notifConf = nil
	if rel != nil && rel.NotificationConfig != 0 {
		conf, err := c.Repo().NotificationConfig().ReadNotificationConfig(rel.NotificationConfig)
		if err != nil {
			c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
			return
		}

		notifConf = conf.ToNotificationConfigType()
	}

	deplNotifier := slack.NewDeploymentNotifier(notifConf, slackInts...)

	notifyOpts := &notifier.NotifyOpts{
		ProjectID:   cluster.ProjectID,
		ClusterID:   cluster.ID,
		ClusterName: cluster.Name,
		Name:        helmRelease.Name,
		Namespace:   helmRelease.Namespace,
		URL: fmt.Sprintf(
			"%s/applications/%s/%s/%s?project_id=%d",
			c.Config().ServerConf.ServerURL,
			url.PathEscape(cluster.Name),
			helmRelease.Namespace,
			helmRelease.Name,
			cluster.ProjectID,
		),
	}

	if upgradeErr != nil {
		notifyOpts.Status = notifier.StatusHelmFailed
		notifyOpts.Info = upgradeErr.Error()

		if !cluster.NotificationsDisabled {
			deplNotifier.Notify(notifyOpts)
		}

		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(
			upgradeErr,
			http.StatusBadRequest,
		))

		return
	}

	if helmRelease.Chart != nil && helmRelease.Chart.Metadata.Name != "job" {
		notifyOpts.Status = notifier.StatusHelmDeployed
		notifyOpts.Version = helmRelease.Version

		if !cluster.NotificationsDisabled {
			deplNotifier.Notify(notifyOpts)
		}
	}

	// update the github actions env if the release exists and is built from source
	if cName := helmRelease.Chart.Metadata.Name; cName == "job" || cName == "web" || cName == "worker" {
		if releaseErr == nil && rel != nil {
			err = baseReleaseHandler.UpdateReleaseRepo(c.Config(), rel, helmRelease)

			if err != nil {
				c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
				return
			}

			gitAction := rel.GitActionConfig

			if gitAction != nil && gitAction.ID != 0 && gitAction.GitlabIntegrationID == 0 {
				gaRunner, err := baseReleaseHandler.GetGARunner(
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

				actionVersion, err := semver.NewVersion(gaRunner.Version)
				if err != nil {
					c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
					return
				}

				if createEnvSecretConstraint.Check(actionVersion) {
					if err := gaRunner.CreateEnvSecret(); err != nil {
						c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
						return
					}
				}
			}
		}
	}
}
