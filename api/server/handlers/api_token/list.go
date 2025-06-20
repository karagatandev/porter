package api_token

import (
	"fmt"
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type APITokenListHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewAPITokenListHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *APITokenListHandler {
	return &APITokenListHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

func (p *APITokenListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proj, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	if !proj.GetFeatureFlag(models.APITokensEnabled, p.Config().LaunchDarklyClient) {
		p.HandleAPIError(w, r, apierrors.NewErrForbidden(fmt.Errorf("api token endpoints are not enabled for this project")))
		return
	}

	tokens, err := p.Repo().APIToken().ListAPITokensByProjectID(proj.ID)
	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	apiTokens := make([]*types.APITokenMeta, 0)

	for _, tok := range tokens {
		apiTokens = append(apiTokens, tok.ToAPITokenMetaType())
	}

	p.WriteResult(w, r, apiTokens)
}
