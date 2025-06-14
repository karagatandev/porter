package user

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/authn"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
)

type UserLogoutHandler struct {
	handlers.PorterHandler
}

func NewUserLogoutHandler(
	config *config.Config,
) *UserLogoutHandler {
	return &UserLogoutHandler{
		PorterHandler: handlers.NewDefaultPorterHandler(config, nil, nil),
	}
}

func (u *UserLogoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := authn.SaveUserUnauthenticated(w, r, u.Config()); err != nil {
		u.HandleAPIError(w, r, apierrors.NewErrInternal(err))
	}

	return
}
