package commonutils

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-github/v41/github"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/requestutils"
	"github.com/karagatandev/porter/api/types"
)

var (
	ErrNoWorkflowRuns   = errors.New("no previous workflow runs found")
	ErrWorkflowNotFound = errors.New("no workflow found, file missing")
)

func GetLatestWorkflowRun(client *github.Client, owner, repo, filename, branch string) (*github.WorkflowRun, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	workflowRuns, ghResponse, err := client.Actions.ListWorkflowRunsByFileName(
		ctx, owner, repo, filename, &github.ListWorkflowRunsOptions{
			Branch: branch,
			ListOptions: github.ListOptions{
				Page:    1,
				PerPage: 1,
			},
		},
	)

	if ghResponse != nil && ghResponse.StatusCode == http.StatusNotFound {
		return nil, ErrWorkflowNotFound
	}

	if err != nil {
		return nil, err
	}

	if workflowRuns == nil || workflowRuns.GetTotalCount() == 0 || len(workflowRuns.WorkflowRuns) == 0 {
		return nil, ErrNoWorkflowRuns
	}

	return workflowRuns.WorkflowRuns[0], nil
}

// GetOwnerAndNameParams gets the owner and name ref for the git repo
func GetOwnerAndNameParams(c handlers.PorterHandler, w http.ResponseWriter, r *http.Request) (string, string, bool) {
	owner, reqErr := requestutils.GetURLParamString(r, types.URLParamGitRepoOwner)

	if reqErr != nil {
		c.HandleAPIError(w, r, reqErr)
		return "", "", false
	}

	name, reqErr := requestutils.GetURLParamString(r, types.URLParamGitRepoName)

	if reqErr != nil {
		c.HandleAPIError(w, r, reqErr)
		return "", "", false
	}

	return owner, name, true
}

// GetBranchParam gets the unencoded branch for the git repo
func GetBranchParam(c handlers.PorterHandler, w http.ResponseWriter, r *http.Request) (string, bool) {
	branch, reqErr := requestutils.GetURLParamString(r, types.URLParamGitBranch)

	if reqErr != nil {
		c.HandleAPIError(w, r, reqErr)
		return "", false
	}

	branch, err := url.QueryUnescape(branch)

	if reqErr != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return "", false
	}

	return branch, true
}
