package environment

import (
	"errors"
	"net/http"

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

type GetDeploymentHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewGetDeploymentHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *GetDeploymentHandler {
	return &GetDeploymentHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

func (c *GetDeploymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ga, _ := r.Context().Value(types.GitInstallationScope).(*integrations.GithubAppInstallation)
	project, _ := r.Context().Value(types.ProjectScope).(*models.Project)
	cluster, _ := r.Context().Value(types.ClusterScope).(*models.Cluster)

	owner, name, ok := commonutils.GetOwnerAndNameParams(c, w, r)

	if !ok {
		return
	}

	request := &types.GetDeploymentRequest{}

	if ok := c.DecodeAndValidate(w, r, request); !ok {
		return
	}

	// read the environment to get the environment id
	env, err := c.Repo().Environment().ReadEnvironment(project.ID, cluster.ID, uint(ga.InstallationID), owner, name)

	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		c.HandleAPIError(w, r, apierrors.NewErrNotFound(errEnvironmentNotFound))
		return
	} else if err != nil {
		c.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	depl, apiErr := validateGetDeploymentRequest(
		project.ID, cluster.ID, env.ID, env.GitRepoOwner, env.GitRepoName, request, c.Repo(),
	)

	if apiErr != nil {
		c.HandleAPIError(w, r, apiErr)
		return
	}

	c.WriteResult(w, r, depl.ToDeploymentType())
}
