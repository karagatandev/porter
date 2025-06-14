package healthcheck

import (
	"net/http"

	"github.com/karagatandev/porter/provisioner/server/config"
)

type LivezHandler struct {
	Config *config.Config
}

func NewLivezHandler(
	config *config.Config,
) *LivezHandler {
	return &LivezHandler{
		Config: config,
	}
}

func (c *LivezHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writeHealthy(w)
}
