package user

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/analytics"
	"github.com/karagatandev/porter/internal/models"
)

type UpdateUserInfoHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewUpdateUserInfoHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *UpdateUserInfoHandler {
	return &UpdateUserInfoHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

func (v *UpdateUserInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(types.UserScope).(*models.User)

	request := &types.UpdateUserInfoRequest{}
	if ok := v.DecodeAndValidate(w, r, request); !ok {
		return
	}

	if request.FirstName != "" && request.LastName != "" && request.CompanyName != "" {
		user.FirstName = request.FirstName
		user.LastName = request.LastName
		user.CompanyName = request.CompanyName
	}

	v.Config().AnalyticsClient.Track(analytics.UserCreateTrack(&analytics.UserCreateTrackOpts{
		UserScopedTrackOpts: analytics.GetUserScopedTrackOpts(user.ID),
		Email:               user.Email,
		FirstName:           user.FirstName,
		LastName:            user.LastName,
		CompanyName:         user.CompanyName,
	}))

	user, err := v.Repo().User().UpdateUser(user)
	if err != nil {
		v.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	v.WriteResult(w, r, user.ToUserType())
}
