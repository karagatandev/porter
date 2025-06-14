//go:build ee
// +build ee

package invite

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type InviteDeleteHandler struct {
	handlers.PorterHandler
	authz.KubernetesAgentGetter
}

func NewInviteDeleteHandler(
	config *config.Config,
) http.Handler {
	return &InviteDeleteHandler{
		PorterHandler:         handlers.NewDefaultPorterHandler(config, nil, nil),
		KubernetesAgentGetter: authz.NewOutOfClusterAgentGetter(config),
	}
}

func (c *InviteDeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	invite, _ := r.Context().Value(types.InviteScope).(*models.Invite)

	if err := c.Repo().Invite().DeleteInvite(invite); err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
	}

	w.WriteHeader(http.StatusOK)
}
