package environment

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v41/github"
	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/commonutils"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/models/integrations"
	"gorm.io/gorm"
)

type FinalizeDeploymentWithErrorsHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewFinalizeDeploymentWithErrorsHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *FinalizeDeploymentWithErrorsHandler {
	return &FinalizeDeploymentWithErrorsHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

func (c *FinalizeDeploymentWithErrorsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ga, _ := r.Context().Value(types.GitInstallationScope).(*integrations.GithubAppInstallation)
	project, _ := r.Context().Value(types.ProjectScope).(*models.Project)
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)

	owner, name, ok := commonutils.GetOwnerAndNameParams(c, w, r)

	if !ok {
		return
	}

	request := &types.FinalizeDeploymentWithErrorsRequest{}

	if ok := c.DecodeAndValidate(w, r, request); !ok {
		return
	}

	if request.Namespace == "" && request.PRNumber == 0 {
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(
			fmt.Errorf("either namespace or pr_number must be present in request body"), http.StatusBadRequest,
		))
		return
	}

	if len(request.Errors) == 0 {
		c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(
			fmt.Errorf("at least one error is required to report"), http.StatusPreconditionFailed,
		))
		return
	}

	var err error

	// read the environment to get the environment id
	env, err := c.Repo().Environment().ReadEnvironment(project.ID, cluster.ID, uint(ga.InstallationID), owner, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.HandleAPIError(w, r, apierrors.NewErrNotFound(errEnvironmentNotFound))
			return
		}

		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	var depl *models.Deployment

	// read the deployment
	if request.PRNumber != 0 {
		depl, err = c.Repo().Environment().ReadDeploymentByGitDetails(env.ID, owner, name, request.PRNumber)

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.HandleAPIError(w, r, apierrors.NewErrNotFound(errDeploymentNotFound))
				return
			}

			c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
			return
		}
	} else if request.Namespace != "" {
		depl, err = c.Repo().Environment().ReadDeployment(env.ID, request.Namespace)

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.HandleAPIError(w, r, apierrors.NewErrNotFound(errDeploymentNotFound))
				return
			}

			c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
			return
		}
	}

	if depl == nil {
		c.HandleAPIError(w, r, apierrors.NewErrNotFound(errDeploymentNotFound))
		return
	}

	client, err := getGithubClientFromEnvironment(c.Config(), env)
	if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	depl.Status = types.DeploymentStatusFailed

	var lastErrors []string

	for resName, errString := range request.Errors {
		lastErrors = append(lastErrors, fmt.Sprintf("%s: %s", resName, errString))
	}

	depl.LastErrors = strings.Join(lastErrors, ",")

	// we do not care of the error in this case because the list deployments endpoint
	// talks to the github API to fetch the deployment status correctly
	c.Repo().Environment().UpdateDeployment(depl)

	// FIXME: ignore the status of this API call for now
	client.Repositories.CreateDeploymentStatus(
		context.Background(), owner, name, depl.GHDeploymentID, &github.DeploymentStatusRequest{
			State:       github.String("failure"),
			Description: github.String("one or more resources failed to build"),
		},
	)

	if !depl.IsBranchDeploy() {
		// add a check for the PR to be open before creating a comment
		prClosed, err := isGithubPRClosed(client, owner, name, int(depl.PullRequestID))
		if err != nil {
			c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusConflict))
			return
		}

		if prClosed {
			c.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(fmt.Errorf("github PR has been closed"),
				http.StatusConflict))
			return
		}

		workflowRun, err := commonutils.GetLatestWorkflowRun(client, depl.RepoOwner, depl.RepoName,
			fmt.Sprintf("porter_%s_env.yml", env.Name), depl.PRBranchFrom)
		if err != nil {
			c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
			return
		}

		commentBody := fmt.Sprintf(
			"## Porter Preview Environments\n"+
				"❌ Errors encountered while deploying the changes\n"+
				"||Deployment Information|\n"+
				"|-|-|\n"+
				"| Latest SHA | [`%s`](https://github.com/%s/%s/commit/%s) |\n"+
				"| Build Logs | %s |\n",
			depl.CommitSHA, depl.RepoOwner, depl.RepoName, depl.CommitSHA, workflowRun.GetHTMLURL(),
		)

		if len(request.SuccessfulResources) > 0 {
			commentBody += "#### Successfully deployed resources\n"

			for _, res := range request.SuccessfulResources {
				if res.ReleaseType == "job" {
					commentBody += fmt.Sprintf("- [`%s`](%s/jobs/%s/%s/%s?project_id=%d)\n",
						res.ReleaseName, c.Config().ServerConf.ServerURL, cluster.Name, depl.Namespace,
						res.ReleaseName, project.ID)
				} else {
					commentBody += fmt.Sprintf("- [`%s`](%s/applications/%s/%s/%s?project_id=%d)\n",
						res.ReleaseName, c.Config().ServerConf.ServerURL, cluster.Name, depl.Namespace,
						res.ReleaseName, project.ID)
				}
			}
		}

		commentBody += "#### Failed resources\n"

		for res, err := range request.Errors {
			commentBody += fmt.Sprintf("<details>\n  <summary><code>%s</code></summary>\n\n  **Error:** %s\n</details>\n", res, err)
		}

		err = createOrUpdateComment(client, c.Repo(), env.NewCommentsDisabled, depl, github.String(commentBody))

		if err != nil {
			c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
			return
		}
	}

	c.WriteResult(w, r, depl.ToDeploymentType())
}
