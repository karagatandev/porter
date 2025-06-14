package gitinstallation

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/karagatandev/porter/internal/telemetry"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/analytics"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/models/integrations"
	"golang.org/x/oauth2"
)

type GithubAppOAuthCallbackHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewGithubAppOAuthCallbackHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *GithubAppOAuthCallbackHandler {
	return &GithubAppOAuthCallbackHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

func (c *GithubAppOAuthCallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.NewSpan(r.Context(), "serve-github-app-oauth-callback")
	defer span.End()

	r = r.Clone(ctx)

	user, _ := r.Context().Value(types.UserScope).(*models.User)

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "user-id", Value: user.ID})

	session, err := c.Config().Store.Get(r, c.Config().ServerConf.CookieName)
	if err != nil {
		err = telemetry.Error(ctx, span, err, "error getting session")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	token, err := c.Config().GithubAppConf.Exchange(oauth2.NoContext, r.URL.Query().Get("code"))
	if err != nil || !token.Valid() {
		telemetry.WithAttributes(span,
			telemetry.AttributeKV{Key: "token-valid", Value: token.Valid()},
			telemetry.AttributeKV{Key: "token-error", Value: err.Error()},
		)
		if redirectStr, ok := session.Values["redirect_uri"].(string); ok && redirectStr != "" {
			telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "redirect-uri", Value: redirectStr})
			// attempt to parse the redirect uri, if it fails just redirect to dashboard
			redirectURI, err := url.Parse(redirectStr)
			if err != nil {
				telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "redirect-uri-parse-error", Value: err.Error()})
				http.Redirect(w, r, "/dashboard", 302)
			}

			http.Redirect(w, r, fmt.Sprintf("%s?%s", redirectURI.Path, redirectURI.RawQuery), 302)
		} else {
			http.Redirect(w, r, "/dashboard", 302)
		}

		return
	}

	oauthInt := &integrations.GithubAppOAuthIntegration{
		SharedOAuthModel: integrations.SharedOAuthModel{
			AccessToken:  []byte(token.AccessToken),
			RefreshToken: []byte(token.RefreshToken),
			Expiry:       token.Expiry,
		},
		UserID: user.ID,
	}

	oauthInt, err = c.Repo().GithubAppOAuthIntegration().CreateGithubAppOAuthIntegration(oauthInt)

	if err != nil {
		err = telemetry.Error(ctx, span, err, "error creating github app oauth integration")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	user.GithubAppIntegrationID = oauthInt.ID

	user, err = c.Repo().User().UpdateUser(user)
	if err != nil {
		err = telemetry.Error(ctx, span, err, "error updating user")
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	c.Config().AnalyticsClient.Track(analytics.GithubConnectionSuccessTrack(
		&analytics.GithubConnectionSuccessTrackOpts{
			UserScopedTrackOpts: analytics.GetUserScopedTrackOpts(user.ID),
		},
	))

	if redirectStr, ok := session.Values["redirect_uri"].(string); ok && redirectStr != "" {
		telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "redirect-uri", Value: redirectStr})
		// attempt to parse the redirect uri, if it fails just redirect to dashboard
		redirectURI, err := url.Parse(redirectStr)
		if err != nil {
			telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "redirect-uri-parse-error", Value: err.Error()})
			http.Redirect(w, r, "/dashboard", 302)
		}

		http.Redirect(w, r, fmt.Sprintf("%s?%s", redirectURI.Path, redirectURI.RawQuery), 302)
	} else {
		http.Redirect(w, r, "/dashboard", 302)
	}
}
