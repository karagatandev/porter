package stack

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type StackListRevisionsHandler struct {
	handlers.PorterHandlerWriter
}

func NewStackListRevisionsHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *StackListRevisionsHandler {
	return &StackListRevisionsHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (p *StackListRevisionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	stack, _ := r.Context().Value(types.StackScope).(*models.Stack)

	res := make([]types.StackRevision, 0)

	for _, stackRev := range stack.Revisions {
		res = append(res, *stackRev.ToStackRevisionType(stack.UID))
	}

	p.WriteResult(w, r, res)
}
