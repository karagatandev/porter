package porter_app

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/handlers/release"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/server/shared/requestutils"
	"github.com/karagatandev/porter/api/types"
	utils "github.com/karagatandev/porter/api/utils/porter_app"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/telemetry"
)

type PorterAppPodsGetHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

func NewPorterAppPodsGetHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *PorterAppPodsGetHandler {
	return &PorterAppPodsGetHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *PorterAppPodsGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cluster, _ := ctx.Value(types.ClusterScope).(*models.Cluster)

	ctx, span := telemetry.NewSpan(ctx, "serve-get-porter-app-pods")
	defer span.End()

	appName, reqErr := requestutils.GetURLParamString(r, types.URLParamPorterAppName)
	if reqErr != nil {
		err := telemetry.Error(ctx, span, reqErr, "error getting stack name from url")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusBadRequest))
		return
	}

	version, reqErr := requestutils.GetURLParamUint(r, types.URLParamReleaseVersion)
	if reqErr != nil {
		err := telemetry.Error(ctx, span, reqErr, "error getting version from url")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusBadRequest))
		return
	}

	namespace := utils.NamespaceFromPorterAppName(appName)
	helmAgent, err := c.GetHelmAgent(ctx, r, cluster, namespace)
	if err != nil {
		err = telemetry.Error(ctx, span, err, "error getting helm agent")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	helmRelease, err := helmAgent.GetRelease(ctx, appName, int(version), false)
	if err != nil {
		err = telemetry.Error(ctx, span, err, "error getting helm release")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	k8sAgent, err := c.GetAgent(r, cluster, namespace)
	if err != nil {
		err = telemetry.Error(ctx, span, err, "error getting k8s agent")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	pods, err := release.GetPodsForRelease(ctx, helmRelease, k8sAgent)
	if err != nil {
		err = telemetry.Error(ctx, span, err, "error getting pods for release")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	c.WriteResult(w, r, pods)
}
