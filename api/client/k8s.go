package client

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/karagatandev/porter/api/types"

	v1 "k8s.io/api/batch/v1"
)

// GetK8sNamespaces gets a namespaces list in a k8s cluster
func (c *Client) GetK8sNamespaces(
	ctx context.Context,
	projectID uint,
	clusterID uint,
) (*types.ListNamespacesResponse, error) {
	resp := &types.ListNamespacesResponse{}

	err := c.getRequest(
		fmt.Sprintf(
			"/projects/%d/clusters/%d/namespaces",
			projectID, clusterID,
		),
		nil,
		resp,
	)

	return resp, err
}

// CreateNewK8sNamespace creates a new namespace in a k8s cluster
func (c *Client) CreateNewK8sNamespace(
	ctx context.Context,
	projectID uint,
	clusterID uint,
	req *types.CreateNamespaceRequest,
) (*types.NamespaceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("invalid request body for creating namespace")
	}

	resp := &types.NamespaceResponse{}

	err := c.postRequest(
		fmt.Sprintf(
			"/projects/%d/clusters/%d/namespaces/create",
			projectID, clusterID,
		),
		req,
		resp,
	)

	return resp, err
}

func (c *Client) GetKubeconfig(
	ctx context.Context,
	projectID uint,
	clusterID uint,
	localKubeconfigPath string,
) (*types.GetTemporaryKubeconfigResponse, error) {
	resp := &types.GetTemporaryKubeconfigResponse{}

	if localKubeconfigPath != "" {
		color.New(color.FgBlue).Fprintf(os.Stderr, "using local kubeconfig: %s\n", localKubeconfigPath)

		if _, err := os.Stat(localKubeconfigPath); !os.IsNotExist(err) {
			file, err := os.Open(localKubeconfigPath)
			if err != nil {
				return nil, err
			}

			data, err := io.ReadAll(file)
			if err != nil {
				return nil, err
			}

			resp.Kubeconfig = append(resp.Kubeconfig, data...)

			return resp, nil
		}
	}

	color.New(color.FgBlue).Fprintln(os.Stderr, "using remote kubeconfig")

	err := c.getRequest(
		fmt.Sprintf(
			"/projects/%d/clusters/%d/kubeconfig",
			projectID, clusterID,
		),
		nil,
		resp,
	)

	if err != nil && strings.Contains(err.Error(), "404") {
		return nil, fmt.Errorf("temporary kubeconfig generation is disabled, please use a local kubeconfig")
	}

	return resp, err
}

func (c *Client) GetEnvGroup(
	ctx context.Context,
	projectID, clusterID uint,
	namespace string,
	req *types.GetEnvGroupRequest,
) (*types.GetEnvGroupResponse, error) {
	resp := &types.GetEnvGroupResponse{}

	err := c.getRequest(
		fmt.Sprintf(
			"/projects/%d/clusters/%d/namespaces/%s/envgroup",
			projectID, clusterID,
			namespace,
		),
		req,
		resp,
	)

	return resp, err
}

func (c *Client) CreateEnvGroup(
	ctx context.Context,
	projectID, clusterID uint,
	namespace string,
	req *types.CreateEnvGroupRequest,
) (*types.EnvGroup, error) {
	resp := &types.EnvGroup{}

	err := c.postRequest(
		fmt.Sprintf(
			"/projects/%d/clusters/%d/namespaces/%s/envgroup/create",
			projectID, clusterID,
			namespace,
		),
		req,
		resp,
	)

	return resp, err
}

func (c *Client) CloneEnvGroup(
	ctx context.Context,
	projectID, clusterID uint,
	namespace string,
	req *types.CloneEnvGroupRequest,
) (*types.EnvGroup, error) {
	resp := &types.EnvGroup{}

	err := c.postRequest(
		fmt.Sprintf(
			"/projects/%d/clusters/%d/namespaces/%s/envgroup/clone",
			projectID, clusterID,
			namespace,
		),
		req,
		resp,
	)

	return resp, err
}

func (c *Client) GetRelease(
	ctx context.Context,
	projectID, clusterID uint,
	namespace, name string,
) (*types.GetReleaseResponse, error) {
	resp := &types.GetReleaseResponse{}

	err := c.getRequest(
		fmt.Sprintf(
			"/projects/%d/clusters/%d/namespaces/%s/releases/%s/0",
			projectID, clusterID,
			namespace, name,
		),
		nil,
		resp,
		withRetryCount(3),
	)

	return resp, err
}

func (c *Client) GetJobs(
	ctx context.Context,
	projectID, clusterID uint,
	namespace, name string,
) ([]v1.Job, error) {
	respArr := make([]v1.Job, 0)

	resp := &respArr

	err := c.getRequest(
		fmt.Sprintf(
			"/projects/%d/clusters/%d/namespaces/%s/releases/%s/0/jobs",
			projectID, clusterID,
			namespace, name,
		),
		nil,
		resp,
	)

	return *resp, err
}

// GetK8sAllPods gets all pods for a given release
func (c *Client) GetK8sAllPods(
	ctx context.Context,
	projectID, clusterID uint,
	namespace, name string,
) (*types.GetReleaseAllPodsResponse, error) {
	resp := &types.GetReleaseAllPodsResponse{}

	err := c.getRequest(
		fmt.Sprintf(
			"/projects/%d/clusters/%d/namespaces/%s/releases/%s/0/pods/all",
			projectID, clusterID,
			namespace, name,
		),
		nil,
		resp,
	)

	return resp, err
}
