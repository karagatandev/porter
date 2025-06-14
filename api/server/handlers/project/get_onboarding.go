package project

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"gorm.io/gorm"
)

type OnboardingGetHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewOnboardingGetHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *OnboardingGetHandler {
	return &OnboardingGetHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

func (p *OnboardingGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proj, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	// look for onboarding
	onboarding, err := p.Repo().Onboarding().ReadProjectOnboarding(proj.ID)
	isNotFound := errors.Is(gorm.ErrRecordNotFound, err)

	if isNotFound {
		p.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(
			fmt.Errorf("project onboarding data not found"),
			http.StatusNotFound,
		))

		return
	} else if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	// return onboarding data type
	p.WriteResult(w, r, onboarding.ToOnboardingType())
}
