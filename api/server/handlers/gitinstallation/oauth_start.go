package gitinstallation

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/analytics"
	"github.com/karagatandev/porter/internal/models"
	"golang.org/x/oauth2"
)

type GithubAppOAuthStartHandler struct {
	handlers.PorterHandler
}

func NewGithubAppOAuthStartHandler(
	config *config.Config,
) *GithubAppOAuthStartHandler {
	return &GithubAppOAuthStartHandler{
		PorterHandler: handlers.NewDefaultPorterHandler(config, nil, nil),
	}
}

func (c *GithubAppOAuthStartHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(types.UserScope).(*models.User)

	c.Config().AnalyticsClient.Track(analytics.GithubConnectionStartTrack(
		&analytics.GithubConnectionStartTrackOpts{
			UserScopedTrackOpts: analytics.GetUserScopedTrackOpts(user.ID),
		},
	))

	http.Redirect(w, r, c.Config().GithubAppConf.AuthCodeURL("", oauth2.AccessTypeOffline), 302)
}
