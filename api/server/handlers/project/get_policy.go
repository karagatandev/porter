package project

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/authz/policy"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type ProjectGetPolicyHandler struct {
	handlers.PorterHandlerWriter
}

func NewProjectGetPolicyHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *ProjectGetPolicyHandler {
	return &ProjectGetPolicyHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (p *ProjectGetPolicyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(types.UserScope).(*models.User)
	proj, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	policyDocLoader := policy.NewBasicPolicyDocumentLoader(p.Config().Repo.Project(), p.Config().Repo.Policy())

	policyDocs, err := policyDocLoader.LoadPolicyDocuments(&policy.PolicyLoaderOpts{
		UserID:    user.ID,
		ProjectID: proj.ID,
	})
	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
	}

	var res types.GetProjectPolicyResponse = policyDocs

	p.WriteResult(w, r, res)
}
