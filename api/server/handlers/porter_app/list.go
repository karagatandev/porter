package porter_app

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type PorterAppListHandler struct {
	handlers.PorterHandlerWriter
}

func NewPorterAppListHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *PorterAppListHandler {
	return &PorterAppListHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (p *PorterAppListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cluster, _ := ctx.Value(types.ClusterScope).(*models.Cluster)

	porterApps, err := p.Repo().PorterApp().ListPorterAppByClusterID(cluster.ID)
	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
		return
	}

	res := make(types.ListPorterAppResponse, 0)

	for _, porterApp := range porterApps {
		res = append(res, porterApp.ToPorterAppType())
	}

	p.WriteResult(w, r, res)
}
