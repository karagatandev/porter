//go:build ee
// +build ee

package credentials

import (
	"net/http"

	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/ee/api/server/handlers/credentials"
)

var NewGetCredentialsHandler func(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) http.Handler

func init() {
	NewGetCredentialsHandler = credentials.NewCredentialsGetHandler
}
