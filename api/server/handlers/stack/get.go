package stack

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

type StackGetHandler struct {
	handlers.PorterHandlerWriter
}

func NewStackGetHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *StackGetHandler {
	return &StackGetHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (p *StackGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	stack, _ := r.Context().Value(types.StackScope).(*models.Stack)

	p.WriteResult(w, r, stack.ToStackType())
}
