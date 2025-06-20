//go:build ee
// +build ee

package invite

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/server/shared/requestutils"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/telemetry"
	"gorm.io/gorm"
)

type InviteAcceptHandler struct {
	handlers.PorterHandler
}

func NewInviteAcceptHandler(
	config *config.Config,
) http.Handler {
	return &InviteAcceptHandler{
		PorterHandler: handlers.NewDefaultPorterHandler(config, nil, nil),
	}
}

func (c *InviteAcceptHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.NewSpan(r.Context(), "serve-invite-accept")
	defer span.End()

	user, _ := ctx.Value(types.UserScope).(*models.User)

	projectID, _ := requestutils.GetURLParamUint(r, types.URLParamProjectID)
	token, _ := requestutils.GetURLParamString(r, types.URLParamInviteToken)

	proj, err := c.Repo().Project().ReadProject(projectID)
	if err != nil {
		vals := url.Values{}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			vals.Add("error", "Invalid invite token")
		} else {
			vals.Add("error", "Unknown error")
		}

		http.Redirect(w, r, fmt.Sprintf("/dashboard?%s", vals.Encode()), 302)
		return
	}

	invite, err := c.Repo().Invite().ReadInviteByToken(token)

	if err != nil || invite.ProjectID != proj.ID {
		vals := url.Values{}
		vals.Add("error", "Invalid invite token")
		http.Redirect(w, r, fmt.Sprintf("/dashboard?%s", vals.Encode()), 302)

		return
	}

	// check that the invite has not expired and has not been accepted
	if invite.IsExpired() || invite.IsAccepted() {
		vals := url.Values{}
		vals.Add("error", "Invite has expired")
		http.Redirect(w, r, fmt.Sprintf("/dashboard?%s", vals.Encode()), 302)

		return
	}

	// check that the invite email matches the user's email
	if user.Email != invite.Email {
		vals := url.Values{}
		vals.Add("error", "Wrong email for invite")
		http.Redirect(w, r, fmt.Sprintf("/dashboard?%s", vals.Encode()), 302)

		return
	}

	kind := invite.Kind

	if kind == "" {
		kind = models.RoleDeveloper
	}

	role := &models.Role{
		Role: types.Role{
			UserID:    user.ID,
			ProjectID: proj.ID,
			Kind:      types.RoleKind(kind),
		},
	}

	if role, err = c.Repo().Project().CreateProjectRole(proj, role); err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	// update the invite
	invite.UserID = user.ID

	if _, err = c.Repo().Invite().UpdateInvite(invite); err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	http.Redirect(w, r, "/dashboard", 302)
}
