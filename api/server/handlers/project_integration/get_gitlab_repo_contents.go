package project_integration

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	ints "github.com/karagatandev/porter/internal/models/integrations"
	"github.com/xanzy/go-gitlab"
)

type GetGitlabRepoContentsHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewGetGitlabRepoContentsHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *GetGitlabRepoContentsHandler {
	return &GetGitlabRepoContentsHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

func (p *GetGitlabRepoContentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	project, _ := r.Context().Value(types.ProjectScope).(*models.Project)
	user, _ := r.Context().Value(types.UserScope).(*models.User)
	gi, _ := r.Context().Value(types.GitlabIntegrationScope).(*ints.GitlabIntegration)

	request := &types.GetGitlabContentsRequest{}

	ok := p.DecodeAndValidate(w, r, request)
	if !ok {
		return
	}

	dir, err := url.QueryUnescape(request.Dir)
	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrForbidden(fmt.Errorf("malformed query param dir")))
		return
	}

	dir = strings.TrimPrefix(dir, "./")

	if len(dir) == 0 {
		dir = "."
	}

	client, err := getGitlabClient(p.Repo(), user.ID, project.ID, gi, p.Config())
	if err != nil {
		if errors.Is(err, errUnauthorizedGitlabUser) {
			p.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(errUnauthorizedGitlabUser, http.StatusUnauthorized))
		}

		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	tree, resp, err := client.Repositories.ListTree(request.RepoPath, &gitlab.ListTreeOptions{
		Path: gitlab.String(dir),
		Ref:  gitlab.String(request.Branch),
	})

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

	var res types.GetContentsResponse

	for _, node := range tree {
		res = append(res, types.GithubDirectoryItem{
			Path: node.Path,
			Type: node.Type,
		})
	}

	p.WriteResult(w, r, res)
}
