package apitest

import (
	"context"
	"net/http"
	"testing"

	"github.com/karagatandev/porter/api/server/authz"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models"
)

func WithProject(t *testing.T, req *http.Request, proj *models.Project) *http.Request {
	ctx := req.Context()
	ctx = context.WithValue(ctx, types.ProjectScope, proj)
	req = req.WithContext(ctx)

	return req
}

func WithRequestScopes(t *testing.T, req *http.Request, reqScopes map[types.PermissionScope]*types.RequestAction) *http.Request {
	ctx := req.Context()
	ctx = authz.NewRequestScopeCtx(ctx, reqScopes)
	req = req.WithContext(ctx)

	return req
}
