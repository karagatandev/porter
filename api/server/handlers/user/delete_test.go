package user_test

import (
	"testing"

	"github.com/karagatandev/porter/api/server/handlers/user"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apitest"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/repository/test"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestDeleteUserSuccessful(t *testing.T) {
	req, rr := apitest.GetRequestAndRecorder(
		t,
		string(types.HTTPVerbDelete),
		"/api/users/current",
		nil,
	)

	config := apitest.LoadConfig(t)
	authUser := apitest.CreateTestUser(t, config, true)
	req = apitest.WithAuthenticatedUser(t, req, authUser)

	handler := user.NewUserDeleteHandler(
		config,
		shared.NewDefaultResultWriter(config.Logger, config.Alerter),
	)

	handler.ServeHTTP(rr, req)

	expUser := &types.CreateUserResponse{
		ID:            1,
		FirstName:     "Mister",
		LastName:      "Porter",
		CompanyName:   "Porter Technologies, Inc.",
		Email:         "mrp@porter.run",
		EmailVerified: true,
	}

	gotUser := &types.CreateUserResponse{}

	apitest.AssertResponseExpected(t, rr, expUser, gotUser)

	// assert that the user has been deleted
	authUser, err := config.Repo.User().ReadUser(1)

	targetErr := gorm.ErrRecordNotFound

	assert.ErrorIs(t, err, targetErr)
}

func TestFailingDeleteUserMethod(t *testing.T) {
	req, rr := apitest.GetRequestAndRecorder(
		t,
		string(types.HTTPVerbDelete),
		"/api/users/current",
		nil,
	)

	config := apitest.LoadConfig(t, test.DeleteUserMethod)
	authUser := apitest.CreateTestUser(t, config, true)
	req = apitest.WithAuthenticatedUser(t, req, authUser)

	handler := user.NewUserDeleteHandler(
		config,
		shared.NewDefaultResultWriter(config.Logger, config.Alerter),
	)

	handler.ServeHTTP(rr, req)

	apitest.AssertResponseInternalServerError(t, rr)
}
