package release

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/helm/grapher"
	"github.com/stefanmcshane/helm/pkg/release"
)

type GetComponentsHandler struct {
	handlers.PorterHandlerReadWriter
	authz.KubernetesAgentGetter
}

func NewGetComponentsHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *GetComponentsHandler {
	return &GetComponentsHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
		KubernetesAgentGetter:   authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *GetComponentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	helmRelease, _ := r.Context().Value(types.ReleaseScope).(*release.Release)

	yamlArr := grapher.ImportMultiDocYAML([]byte(helmRelease.Manifest))
	objects := grapher.ParseObjs(yamlArr, helmRelease.Namespace)

	parsed := grapher.ParsedObjs{
		Objects: objects,
	}

	parsed.GetControlRel()
	parsed.GetLabelRel()
	parsed.GetSpecRel()

	c.WriteResult(w, r, parsed)
}
