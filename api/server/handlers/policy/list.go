package policy

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type PolicyListHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewPolicyListHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *PolicyListHandler {
	return &PolicyListHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

func (p *PolicyListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proj, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	policies, err := p.Repo().Policy().ListPoliciesByProjectID(proj.ID)
	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	res := make([]*types.APIPolicyMeta, 0)

	for _, policy := range policies {
		res = append(res, policy.ToAPIPolicyTypeMeta())
	}

	p.WriteResult(w, r, res)
}
