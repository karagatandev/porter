package project_integration

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
	ints "github.com/karagatandev/porter/internal/models/integrations"
	"github.com/xanzy/go-gitlab"
)

type ListGitlabRepoBranchesHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewListGitlabRepoBranchesHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *ListGitlabRepoBranchesHandler {
	return &ListGitlabRepoBranchesHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

func (p *ListGitlabRepoBranchesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	project, _ := r.Context().Value(types.ProjectScope).(*models.Project)
	user, _ := r.Context().Value(types.UserScope).(*models.User)
	gi, _ := r.Context().Value(types.GitlabIntegrationScope).(*ints.GitlabIntegration)

	request := &types.ListGitlabRepoBranchesRequest{}
	if ok := p.DecodeAndValidate(w, r, request); !ok {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(errors.New("cannot decode and validate request")))
		return
	}

	repoPath := request.RepoPath

	client, err := getGitlabClient(p.Repo(), user.ID, project.ID, gi, p.Config())
	if err != nil {
		if errors.Is(err, errUnauthorizedGitlabUser) {
			p.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(errUnauthorizedGitlabUser, http.StatusUnauthorized))
		}

		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	branches, resp, err := client.Branches.ListBranches(repoPath,
		&gitlab.ListBranchesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    1,
				PerPage: 20,
			},
			Search: &request.SearchTerm,
		},
	)

	if resp.StatusCode == http.StatusUnauthorized {
		p.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(fmt.Errorf("unauthorized gitlab user"), http.StatusUnauthorized))
		return
	} else if resp.StatusCode == http.StatusNotFound {
		p.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(fmt.Errorf("no such gitlab project found"), http.StatusNotFound))
		return
	}

	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	var res []string

	for _, branch := range branches {
		res = append(res, branch.Name)
	}

	p.WriteResult(w, r, res)
}
