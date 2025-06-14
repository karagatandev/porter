//go:build ee
// +build ee

package invite

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type InviteUpdateRoleHandler struct {
	handlers.PorterHandlerReader
}

func NewInviteUpdateRoleHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
) http.Handler {
	return &InviteUpdateRoleHandler{
		PorterHandlerReader: handlers.NewDefaultPorterHandler(config, decoderValidator, nil),
	}
}

func (c *InviteUpdateRoleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	invite, _ := r.Context().Value(types.InviteScope).(*models.Invite)

	request := &types.UpdateInviteRoleRequest{}

	if ok := c.DecodeAndValidate(w, r, request); !ok {
		return
	}

	invite.Kind = request.Kind

	if _, err := c.Repo().Invite().UpdateInvite(invite); err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
	}

	w.WriteHeader(http.StatusOK)
}
