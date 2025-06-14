package user_test

import (
	"testing"

	"github.com/karagatandev/porter/api/server/handlers/user"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apitest"
	"github.com/karagatandev/porter/api/types"
)

func TestGetCurrentUserSuccessful(t *testing.T) {
	config := apitest.LoadConfig(t)
	authUser := apitest.CreateTestUser(t, config, true)
	req, rr := apitest.GetRequestAndRecorder(t, string(types.HTTPVerbPost), "/api/auth/check", nil)

	req = apitest.WithAuthenticatedUser(t, req, authUser)

	handler := user.NewUserGetCurrentHandler(
		config,
		shared.NewDefaultResultWriter(config.Logger, config.Alerter),
	)

	handler.ServeHTTP(rr, req)

	expUser := &types.GetAuthenticatedUserResponse{
		ID:            1,
		FirstName:     "Mister",
		LastName:      "Porter",
		CompanyName:   "Porter Technologies, Inc.",
		Email:         "mrp@porter.run",
		EmailVerified: true,
	}

	gotUser := &types.GetAuthenticatedUserResponse{}

	apitest.AssertResponseExpected(t, rr, expUser, gotUser)
}
