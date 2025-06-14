package state

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/provisioner/integrations/storage"
	"github.com/karagatandev/porter/provisioner/server/config"

	ptypes "github.com/karagatandev/porter/provisioner/types"
)

type StateGetHandler struct {
	Config *config.Config
}

func NewStateGetHandler(
	config *config.Config,
) *StateGetHandler {
	return &StateGetHandler{
		Config: config,
	}
}

func (c *StateGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// read the infra from the attached scope
	infra, _ := r.Context().Value(types.InfraScope).(*models.Infra)

	fileBytes, err := c.Config.StorageManager.ReadFile(infra, ptypes.DefaultCurrentStateFile, true)
	if err != nil {
		// if the file does not exist yet, return a 404 status code
		if errors.Is(err, storage.FileDoesNotExist) {
			apierrors.HandleAPIError(c.Config.Logger, c.Config.Alerter, w, r, apierrors.NewErrPassThroughToClient(
				fmt.Errorf("current state file does not exist yet"),
				http.StatusNotFound,
			), true)

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
