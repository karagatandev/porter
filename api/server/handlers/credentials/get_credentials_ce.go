//go:build !ee
// +build !ee

package credentials

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/config"
)

type GetCredentialsHandler struct {
	handlers.PorterHandlerReader
	handlers.Unavailable
}

func NewGetCredentialsHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) http.Handler {
	return handlers.NewUnavailable(config, "get_credential")
}
