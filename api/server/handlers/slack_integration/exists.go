package slack_integration

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/shared/apierrors"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type SlackIntegrationExists struct {
	handlers.PorterHandler
}

func NewSlackIntegrationExists(
	config *config.Config,
) *SlackIntegrationExists {
	return &SlackIntegrationExists{
		PorterHandler: handlers.NewDefaultPorterHandler(config, nil, nil),
	}
}

func (p *SlackIntegrationExists) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	project, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	slackInts, err := p.Repo().SlackIntegration().ListSlackIntegrationsByProjectID(project.ID)
	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	if len(slackInts) != 0 {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
