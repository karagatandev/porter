package user

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/authn"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type UserDeleteHandler struct {
	handlers.PorterHandlerWriter
}

func NewUserDeleteHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *UserDeleteHandler {
	return &UserDeleteHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (u *UserDeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(types.UserScope).(*models.User)

	user, err := u.Repo().User().DeleteUser(user)
	if err != nil {
		u.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	// set the user as unauthenticated in the session
	if err := authn.SaveUserUnauthenticated(w, r, u.Config()); err != nil {
		u.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	u.WriteResult(w, r, user.ToUserType())
}
