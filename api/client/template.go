package client

import (
	"context"
	"fmt"

	"github.com/karagatandev/porter/api/types"
)

func (c *Client) ListTemplates(
	ctx context.Context,
	projectID uint,
	req *types.ListTemplatesRequest,
) (*types.ListTemplatesResponse, error) {
	resp := &types.ListTemplatesResponse{}

	err := c.getRequest(
		fmt.Sprintf(
			"/v1/projects/%d/templates", projectID,
		),
		req,
		resp,
	)

	return resp, err
}

func (c *Client) GetTemplate(
	ctx context.Context,
	projectID uint,
	name, version string,
	req *types.GetTemplateRequest,
) (*types.GetTemplateResponse, error) {
	resp := &types.GetTemplateResponse{}

	err := c.getRequest(
		fmt.Sprintf(
			"/v1/projects/%d/templates/%s/versions/%s",
			projectID,
			name, version,
		),
		req,
		resp,
	)

	return resp, err
}
