package gitinstallation

import (
	"context"
	"net/http"
	"sync"

	"github.com/google/go-github/v41/github"
	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
)

type GithubListReposHandler struct {
	handlers.PorterHandlerWriter
	authz.KubernetesAgentGetter
}

func NewGithubListReposHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *GithubListReposHandler {
	return &GithubListReposHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (c *GithubListReposHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	client, err := GetGithubAppClientFromRequest(c.Config(), r)
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	// figure out number of repositories
	opt := &github.ListOptions{
		PerPage: 100,
	}

	repoList, resp, err := client.Apps.ListRepos(context.Background(), opt)
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	allRepos := repoList.Repositories

	// make workers to get pages concurrently
	const WCOUNT = 5
	numPages := resp.LastPage + 1
	var workerErr error
	var mu sync.Mutex
	var wg sync.WaitGroup

	worker := func(cp int) {
		defer wg.Done()

		for cp < numPages {
			cur_opt := &github.ListOptions{
				Page:    cp,
				PerPage: 100,
			}

			repos, _, err := client.Apps.ListRepos(context.Background(), cur_opt)
			if err != nil {
				mu.Lock()
				workerErr = err
				mu.Unlock()
				return
			}

			mu.Lock()
			allRepos = append(allRepos, repos.Repositories...)
			mu.Unlock()

			cp += WCOUNT
		}
	}

	var numJobs int
	if numPages > WCOUNT {
		numJobs = WCOUNT
	} else {
		numJobs = numPages
	}

	wg.Add(numJobs)

	// page 1 is already loaded so we start with 2
	for i := 1; i <= numJobs; i++ {
		go worker(i + 1)
	}

	wg.Wait()

	if workerErr != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	res := make(types.ListReposResponse, 0)

	for _, repo := range allRepos {
		res = append(res, types.Repo{
			FullName: repo.GetFullName(),
			Kind:     "github",
		})
	}

	c.WriteResult(w, r, res)
}
