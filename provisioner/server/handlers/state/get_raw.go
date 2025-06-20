package state

import (
	"errors"
	"net/http"

	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/provisioner/integrations/storage"
	"github.com/karagatandev/porter/provisioner/server/config"
	ptypes "github.com/karagatandev/porter/provisioner/types"
)

type RawStateGetHandler struct {
	Config *config.Config
}

func NewRawStateGetHandler(
	config *config.Config,
) *RawStateGetHandler {
	return &RawStateGetHandler{
		Config: config,
	}
}

func (c *RawStateGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// read the infra from the attached scope
	infra, _ := r.Context().Value(types.InfraScope).(*models.Infra)

	fileBytes, err := c.Config.StorageManager.ReadFile(infra, ptypes.DefaultTerraformStateFile, true)
	if err != nil {
		// if the file does not exist yet, just return an empty body with a 200-response code
		if errors.Is(err, storage.FileDoesNotExist) {
			return
		}

		apierrors.HandleAPIError(c.Config.Logger, c.Config.Alerter, w, r, apierrors.NewErrInternal(err), true)
		return
	}

	if _, err = w.Write(fileBytes); err != nil {
		apierrors.HandleAPIError(c.Config.Logger, c.Config.Alerter, w, r, apierrors.NewErrInternal(err), true)

		return
	}
}
