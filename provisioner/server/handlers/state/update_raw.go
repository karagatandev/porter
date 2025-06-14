package state

import (
	"io"
	"net/http"

	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/provisioner/server/config"

	ptypes "github.com/karagatandev/porter/provisioner/types"
)

type RawStateUpdateHandler struct {
	Config *config.Config
}

func NewRawStateUpdateHandler(
	config *config.Config,
) *RawStateUpdateHandler {
	return &RawStateUpdateHandler{
		Config: config,
	}
}

func (c *RawStateUpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// read the infra from the attached scope
	infra, _ := r.Context().Value(types.InfraScope).(*models.Infra)

	// read state file
	fileBytes, err := io.ReadAll(r.Body)
	if err != nil {
		apierrors.HandleAPIError(c.Config.Logger, c.Config.Alerter, w, r, apierrors.NewErrInternal(err), true)

		return
	}

	err = c.Config.StorageManager.WriteFile(infra, ptypes.DefaultTerraformStateFile, fileBytes, true)

	if err != nil {
		apierrors.HandleAPIError(c.Config.Logger, c.Config.Alerter, w, r, apierrors.NewErrInternal(err), true)

		return
	}

	return
}
