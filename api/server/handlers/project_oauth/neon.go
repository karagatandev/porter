package project_oauth

import (
	"net/http"

	"github.com/karagatandev/porter/internal/telemetry"

	"golang.org/x/oauth2"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/internal/oauth"
)

// ProjectOAuthNeonHandler is the handler which redirects to the neon oauth flow
type ProjectOAuthNeonHandler struct {
	handlers.PorterHandlerReadWriter
}

// NewProjectOAuthNeonHandler generates a new ProjectOAuthNeonHandler
func NewProjectOAuthNeonHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *ProjectOAuthNeonHandler {
	return &ProjectOAuthNeonHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

// ServeHTTP populates the oauth session with state and project id then redirects the user to the neon oauth flow
func (p *ProjectOAuthNeonHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.NewSpan(r.Context(), "serve-project-oauth-neon")
	defer span.End()

	r = r.Clone(ctx)

	state := oauth.CreateRandomState()

	if err := p.PopulateOAuthSession(ctx, w, r, state, true, false, "", 0); err != nil {
		err = telemetry.Error(ctx, span, err, "population oauth session failed")
		p.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	url := p.Config().NeonConf.AuthCodeURL(state, oauth2.AccessTypeOffline)

	http.Redirect(w, r, url, http.StatusFound)
}
