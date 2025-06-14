package metadata

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/config"
)

type MetadataGetHandler struct {
	handlers.PorterHandlerWriter
}

func NewMetadataGetHandler(
	config *config.Config,
	writer shared.ResultWriter,
) *MetadataGetHandler {
	return &MetadataGetHandler{
		PorterHandlerWriter: handlers.NewDefaultPorterHandler(config, nil, writer),
	}
}

func (v *MetadataGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v.WriteResult(w, r, v.Config().Metadata)
}
