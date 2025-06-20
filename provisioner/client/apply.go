package client

import (
	"context"
	"fmt"

	"github.com/karagatandev/porter/api/types"
	ptypes "github.com/karagatandev/porter/provisioner/types"
)

// Apply initiates a new apply operation for infra
func (c *Client) Apply(
	ctx context.Context,
	projID, infraID uint,
	req *ptypes.ApplyBaseRequest,
) (*types.Operation, error) {
	resp := &types.Operation{}

	err := c.postRequest(
		fmt.Sprintf(
			"/projects/%d/infras/%d/apply",
			projID,
			infraID,
		),
		req,
		resp,
	)

	return resp, err
}
