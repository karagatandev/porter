package api_token

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/server/shared/requestutils"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"gorm.io/gorm"
)

type APITokenRevokeHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewAPITokenRevokeHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *APITokenRevokeHandler {
	return &APITokenRevokeHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

func (p *APITokenRevokeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proj, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	if !proj.GetFeatureFlag(models.APITokensEnabled, p.Config().LaunchDarklyClient) {
		p.HandleAPIError(w, r, apierrors.NewErrForbidden(fmt.Errorf("api token endpoints are not enabled for this project")))
		return
	}

	// get the token id from the request
	tokenID, reqErr := requestutils.GetURLParamString(r, types.URLParamTokenID)

	if reqErr != nil {
		p.HandleAPIError(w, r, reqErr)
		return
	}

	token, err := p.Repo().APIToken().ReadAPIToken(proj.ID, tokenID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			p.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(
				fmt.Errorf("token with id %s not found in project", tokenID),
				http.StatusNotFound,
			))
			return
		}

		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	token.Revoked = true

	token, err = p.Repo().APIToken().UpdateAPIToken(token)

	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	p.WriteResult(w, r, token.ToAPITokenMetaType())
}
