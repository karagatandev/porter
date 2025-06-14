package project

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/usage"
)

type ProjectGetUsageHandler struct {
	handlers.PorterHandlerWriter
}

func NewProjectGetUsageHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *ProjectGetUsageHandler {
	return &ProjectGetUsageHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (p *ProjectGetUsageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proj, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	if !p.Config().ServerConf.UsageTrackingEnabled {
		p.WriteResult(w, r, &types.GetProjectUsageResponse{
			Limit:      types.EnterprisePlan,
			IsExceeded: false,
		})

		return
	}

	res := &types.GetProjectUsageResponse{}

	currUsage, limit, usageCache, err := usage.GetUsage(&usage.GetUsageOpts{
		Project:                          proj,
		DOConf:                           p.Config().DOConf,
		Repo:                             p.Repo(),
		WhitelistedUsers:                 p.Config().WhitelistedUsers,
		ClusterControlPlaneServiceClient: p.Config().ClusterControlPlaneClient,
	})
	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	res.Current = *currUsage
	res.Limit = *limit
	res.IsExceeded = usageCache.Exceeded
	res.ExceededSince = usageCache.ExceededSince

	p.WriteResult(w, r, res)
}
