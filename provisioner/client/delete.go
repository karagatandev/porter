package client

import (
	"context"
	"fmt"

	"github.com/karagatandev/porter/api/types"
	ptypes "github.com/karagatandev/porter/provisioner/types"
)

// Apply initiates a new apply operation for infra
func (c *Client) Delete(
	ctx context.Context,
	projID, infraID uint,
	req *ptypes.DeleteBaseRequest,
) (*types.Operation, error) {
	resp := &types.Operation{}

	err := c.deleteRequest(
		fmt.Sprintf(
			"/projects/%d/infras/%d",
			projID,
			infraID,
		),
		req,
		resp,
	)

	return resp, err
}
