package release

import (
	"net/http"

	semver "github.com/Masterminds/semver/v3"
	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/helm/loader"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/telemetry"
	"github.com/karagatandev/porter/internal/templater/parser"
	"github.com/stefanmcshane/helm/pkg/release"
	"gorm.io/gorm"
)

type ReleaseGetHandler struct {
	handlers.PorterHandlerWriter
	authz.KubernetesAgentGetter
}

func NewReleaseGetHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *ReleaseGetHandler {
	return &ReleaseGetHandler{
		PorterHandlerWriter:   handlers.NewDefaultPorterHandler(config, nil, writer),
		KubernetesAgentGetter: authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *ReleaseGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.NewSpan(r.Context(), "serve-get-release")
	defer span.End()

	helmRelease, _ := ctx.Value(types.ReleaseScope).(*release.Release)
	cluster, _ := ctx.Value(types.ClusterScope).(*models.Cluster)

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "release-name", Value: helmRelease.Name})

	res := &types.Release{
		Release: helmRelease,
	}

	// look up the release in the database; if not found, do not populate Porter fields
	release, err := c.Repo().Release().ReadRelease(cluster.ID, helmRelease.Name, helmRelease.Namespace)
	if err == nil {
		res.PorterRelease = release.ToReleaseType()

		res.ID = release.ID
		res.WebhookToken = release.WebhookToken

		if release.GitActionConfig != nil {
			res.GitActionConfig = release.GitActionConfig.ToGitActionConfigType()
		}

		if release.BuildConfig != 0 {
			bc, err := c.Repo().BuildConfig().GetBuildConfig(release.BuildConfig)
			if err != nil {
				err = telemetry.Error(ctx, span, err, "unable to get build config")
				c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
				return
			}

			res.BuildConfig = bc.ToBuildConfigType()
		}

		if release.StackResourceID != 0 {
			stackResource, err := c.Repo().Stack().ReadStackResource(release.StackResourceID)
			if err != nil {
				err = telemetry.Error(ctx, span, err, "unable to get stack resource")
				c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
				return
			}

			stackRevision, err := c.Repo().Stack().ReadStackRevision(stackResource.StackRevisionID)
			if err != nil {
				err = telemetry.Error(ctx, span, err, "unable to get stack revision")
				c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
				return
			}

			stack, err := c.Repo().Stack().ReadStackByID(cluster.ProjectID, stackRevision.StackID)
			if err != nil {
				err = telemetry.Error(ctx, span, err, "unable to get stack")
				c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
				return
			}

			res.StackID = stack.UID
		}
	} else if err != gorm.ErrRecordNotFound {
		err = telemetry.Error(ctx, span, err, "unable to get release")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	} else {
		res.PorterRelease = &types.PorterRelease{}
	}

	// detect if Porter application chart and attempt to get the latest version
	// from chart repo
	cache := c.Config().URLCache
	chartRepoURL, foundFirst := cache.GetURL(helmRelease.Chart.Metadata.Name)

	if !foundFirst {
		cache.Update()

		chartRepoURL, _ = cache.GetURL(helmRelease.Chart.Metadata.Name)
	}

	if chartRepoURL != "" {
		repoIndex, err := loader.LoadRepoIndexPublic(chartRepoURL)

		if err == nil {
			porterChart := loader.FindPorterChartInIndexList(repoIndex, res.Chart.Metadata.Name)
			res.LatestVersion = res.Chart.Metadata.Version

			// set latest version to the greater of porterChart.Versions and res.Chart.Metadata.Version
			porterChartVersion, porterChartErr := semver.NewVersion(porterChart.Versions[0])
			currChartVersion, currChartErr := semver.NewVersion(res.Chart.Metadata.Version)

			if currChartErr == nil && porterChartErr == nil && porterChartVersion.GreaterThan(currChartVersion) {
				res.LatestVersion = porterChart.Versions[0]
			}
		}
	}

	// look for the form using the dynamic client
	dynClient, err := c.GetDynamicClient(r, cluster)
	if err != nil {
		err = telemetry.Error(ctx, span, err, "unable to get dynamic client")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	parserDef := &parser.ClientConfigDefault{
		DynamicClient: dynClient,
		HelmChart:     helmRelease.Chart,
		HelmRelease:   helmRelease,
	}

	form, err := parser.GetFormFromRelease(parserDef, helmRelease)

	if err != nil {
		c.HandleAPIErrorNoWrite(w, r, apierrors.NewErrInternal(err))
	} else {
		res.Form = form
	}
	// if form not populated, detect common charts
	if res.Form == nil {
		// for now just case by name
		if res.Release.Chart.Name() == "cert-manager" {
			formYAML, err := parser.FormYAMLFromBytes(parserDef, []byte(certManagerForm), "", "")

			if err == nil {
				res.Form = formYAML
			}
		} else if res.Release.Chart.Name() == "velero" {
			formYAML, err := parser.FormYAMLFromBytes(parserDef, []byte(veleroForm), "", "")

			if err == nil {
				res.Form = formYAML
			}
		}
	}

	c.WriteResult(w, r, res)
}

const certManagerForm string = `tags:
- hello
tabs:
- name: main
  context:
    type: cluster
    config:
      group: cert-manager.io
      version: v1
      resource: certificates
  label: Certificates
  sections:
  - name: section_one
    contents:
    - type: heading
      label: Certificates
    - type: resource-list
      settings:
        options:
          resource-button:
            name: "Renew Certificate"
            description: "This will delete the existing certificate resource, triggering a new certificate request."
            actions:
            - delete:
                scope: namespace
                relative_uri: /crd
                context:
                  type: cluster
                  config:
                    group: cert-manager.io
                    version: v1
                    resource: certificates
      value: |
        .items[] | {
          metadata: .metadata,
          name: "\(.spec.dnsNames | join(","))",
          label: "\(.metadata.namespace)/\(.metadata.name)",
          status: (
            ([.status.conditions[].type] | index("Ready")) as $index | (
              if $index then (
                if .status.conditions[$index].status == "True" then "Ready" else "Not Ready" end
              ) else (
                "Not Ready"
              ) end
            )
          ),
          timestamp: .status.conditions[0].lastTransitionTime,
          message: [.status.conditions[].message] | unique | join(","),
          data: {}
        }`

const veleroForm string = `tags:
- hello
tabs:
- name: main
  context:
    type: cluster
    config:
      group: velero.io
      version: v1
      resource: backups
  label: Backups
  sections:
  - name: section_one
    contents: 
    - type: heading
      label: 💾 Velero Backups
    - type: resource-list
      value: |
        .items[] | { 
          name: .metadata.name, 
          label: .metadata.namespace,
          status: .status.phase,
          timestamp: .status.completionTimestamp,
          message: [
            (if .status.volumeSnapshotsAttempted then "\(.status.volumeSnapshotsAttempted) volume snapshots attempted, \(.status.volumeSnapshotsCompleted) completed." else null end),
            "Finished \(.status.completionTimestamp).",
            "Backup expires on \(.status.expiration)."
          ]|join(" "),
          data: {
            "Included Namespaces": (if .spec.includedNamespaces then .spec.includedNamespaces|join(",") else "* (all)" end),
            "Included Resources": (if .spec.includedResources then .spec.includedResources|join(",") else "* (all)" end),
            "Storage Location": .spec.storageLocation
          }
        }`
