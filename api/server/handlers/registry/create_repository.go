package registry

import (
	"net/http"
	"strings"

	"github.com/karagatandev/porter/api/server/handlers"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apierrors"
	"github.com/karagatandev/porter/api/server/shared/config"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/registry"
	"github.com/karagatandev/porter/internal/telemetry"
)

type RegistryCreateRepositoryHandler struct {
	handlers.PorterHandlerReadWriter
}

func NewRegistryCreateRepositoryHandler(
	config *config.Config,
	decoderValidator shared.RequestDecoderValidator,
	writer shared.ResultWriter,
) *RegistryCreateRepositoryHandler {
	return &RegistryCreateRepositoryHandler{
		PorterHandlerReadWriter: handlers.NewDefaultPorterHandler(config, decoderValidator, writer),
	}
}

func (p *RegistryCreateRepositoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reg, _ := ctx.Value(types.RegistryScope).(*models.Registry)
	ctx, span := telemetry.NewSpan(r.Context(), "serve-create-repository")
	defer span.End()

	request := &types.CreateRegistryRepositoryRequest{}

	ok := p.DecodeAndValidate(w, r, request)
	if !ok {
		err := telemetry.Error(ctx, span, nil, "error decoding request")
		p.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusBadRequest))
		return
	}

	_reg := registry.Registry(*reg)
	regAPI := &_reg

	// parse the name from the registry
	nameSpl := strings.Split(request.ImageRepoURI, "/")
	sanitizedName := strings.ReplaceAll(strings.ReplaceAll(nameSpl[len(nameSpl)-1], "_", "-"), ".", "-")
	repoName := strings.ToLower(sanitizedName)
	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "repo-name", Value: repoName},
		telemetry.AttributeKV{Key: "registry-id", Value: reg.ID},
		telemetry.AttributeKV{Key: "image-repo-uri", Value: request.ImageRepoURI},
	)

	err := regAPI.CreateRepository(ctx, p.Config(), repoName)
	if err != nil {
		err = telemetry.Error(ctx, span, err, "error creating repository")
		p.HandleAPIError(w, r, apierrors.NewErrPassThroughToClient(err, http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusCreated)
}
