package gitinstallation

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models/integrations"
)

type GitInstallationGetHandler struct {
	handlers.PorterHandlerWriter
}

func NewGitInstallationGetHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *GitInstallationGetHandler {
	return &GitInstallationGetHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (c *GitInstallationGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ga, _ := r.Context().Value(types.GitInstallationScope).(*integrations.GithubAppInstallation)

	c.WriteResult(w, r, ga.ToGitInstallationType())
}
