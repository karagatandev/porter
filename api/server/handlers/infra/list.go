package infra

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type InfraListHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewInfraListHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *InfraListHandler {
	return &InfraListHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

func (p *InfraListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proj, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	req := &types.ListInfraRequest{}

	if ok := p.DecodeAndValidate(w, r, req); !ok {
		return
	}

	infras, err := p.Repo().Infra().ListInfrasByProjectID(proj.ID, req.Version)
	if err != nil {
		p.HandleAPIError(w, r, apierrors.NewErrInternal(err))
	}

	infraList := make([]*types.Infra, 0)

	for _, infra := range infras {
		infraList = append(infraList, infra.ToInfraType())
	}

	var res types.ListProjectInfraResponse = infraList

	p.WriteResult(w, r, res)
}
