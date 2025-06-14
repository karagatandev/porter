package project_test

import (
	"testing"

	"github.com/karagatandev/porter/api/server/handlers/project"
	"github.com/karagatandev/porter/api/server/shared"
	"github.com/karagatandev/porter/api/server/shared/apitest"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/features"
	"github.com/karagatandev/porter/internal/models"
)

func TestGetProjectSuccessful(t *testing.T) {
	// create a test project
	config := apitest.LoadConfig(t)
	user := apitest.CreateTestUser(t, config, true)
	proj, _, err := project.CreateProjectWithUser(config.Repo.Project(), &models.Project{
		Name: "test-project",
	}, user)
	if err != nil {
		t.Fatal(err)
	}

	req, rr := apitest.GetRequestAndRecorder(t, string(types.HTTPVerbPost), "/api/projects/1", nil)

	req = apitest.WithAuthenticatedUser(t, req, user)
	req = apitest.WithProject(t, req, proj)

	handler := project.NewProjectGetHandler(
		config,
		shared.NewDefaultResultWriter(config.Logger, config.Alerter),
	)

	handler.ServeHTTP(rr, req)

	expProject := proj.ToProjectType(&features.Client{})
	gotProject := types.Project{}

	apitest.AssertResponseExpected(t, rr, &expProject, &gotProject)
}
