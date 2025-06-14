package template

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/helm/loader"
)

type TemplateListHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewTemplateListHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *TemplateListHandler {
	return &TemplateListHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

func (t *TemplateListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	request := &types.ListTemplatesRequest{}

	ok := t.DecodeAndValidate(w, r, request)

	if !ok {
		return
	}

	repoURL := request.RepoURL

	if repoURL == "" {
		repoURL = t.Config().ServerConf.DefaultApplicationHelmRepoURL
	}

	repoIndex, err := loader.LoadRepoIndexPublic(repoURL)
	if err != nil {
		t.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	porterCharts := loader.RepoIndexToPorterChartList(repoIndex, repoURL)

	t.WriteResult(w, r, porterCharts)
}
