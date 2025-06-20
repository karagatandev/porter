package models

import (
	"encoding/json"

	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/internal/models/integrations"
	"gorm.io/gorm"
)

// ClusterAuth is an auth mechanism that a cluster candidate can resolve
type ClusterAuth string

// The support cluster candidate auth mechanisms
const (
	X509      ClusterAuth = "x509"
	Basic     ClusterAuth = "basic"
	Bearer    ClusterAuth = "bearerToken"
	OIDC      ClusterAuth = "oidc"
	GCP       ClusterAuth = "gcp-sa"
	AWS       ClusterAuth = "aws-sa"
	DO        ClusterAuth = "do-oauth"
	Azure     ClusterAuth = "azure-sp"
	Local     ClusterAuth = "local"
	InCluster ClusterAuth = "in-cluster"
)

// Cluster is an integration that can connect to a Kubernetes cluster via
// a specific auth mechanism
type Cluster struct {
	gorm.Model

	// The auth mechanism that this cluster will use
	AuthMechanism ClusterAuth `json:"auth_mechanism"`

	// The project that this integration belongs to
	ProjectID uint `json:"project_id"`

	// Whether or not the Porter agent integration is enabled on the cluster
	AgentIntegrationEnabled bool

	// Name of the cluster
	Name string `json:"name"`

	// VanityName allows for a display-only name without changing how the cluster looks
	VanityName string `json:"vanity_name"`

	// Server endpoint for the cluster
	Server string `json:"server"`

	// Additional fields optionally used by the kube client
	ClusterLocationOfOrigin string `json:"location_of_origin,omitempty"`
	TLSServerName           string `json:"tls-server-name,omitempty"`
	InsecureSkipTLSVerify   bool   `json:"insecure-skip-tls-verify,omitempty"`
	ProxyURL                string `json:"proxy-url,omitempty"`
	UserLocationOfOrigin    string
	UserImpersonate         string `json:"act-as,omitempty"`
	UserImpersonateGroups   string `json:"act-as-groups,omitempty"`

	InfraID uint `json:"infra_id"`

	NotificationsDisabled bool `json:"notifications_disabled"`

	PreviewEnvsEnabled bool

	AWSClusterID string

	// Status defines the current status of the cluster. Accepted values: [READY, UPDATING]
	Status types.ClusterStatus `json:"status"`

	// ProvisionedBy is used for identifing the provisioner used for the cluster. Accepted values: [CAPI, ]
	ProvisionedBy string `json:"provisioned_by"`

	// CloudProvider is the cloud provider that hosts the Kubernetes Cluster. Accepted values: [AWS, GCP, AZURE]
	CloudProvider string `json:"cloud_provider"`

	// CloudProviderCredentialIdentifier is a reference to find the credentials required for access the cluster's API.
	// This was likely the credential that was used to create the cluster.
	// For AWS EKS clusters, this will be an ARN for the final target role in the assume role chain.
	CloudProviderCredentialIdentifier string `json:"cloud_provider_credential_identifier"`

	// ------------------------------------------------------------------
	// All fields below this line are encrypted before storage
	// ------------------------------------------------------------------

	// The various auth mechanisms available to the integration
	KubeIntegrationID  uint
	OIDCIntegrationID  uint
	GCPIntegrationID   uint
	AWSIntegrationID   uint
	DOIntegrationID    uint
	AzureIntegrationID uint

	// A token cache that can be used by an auth mechanism, if desired
	TokenCache   integrations.ClusterTokenCache `json:"token_cache" gorm:"-" sql:"-"`
	TokenCacheID uint                           `gorm:"token_cache_id"`

	// CertificateAuthorityData for the cluster, encrypted at rest
	CertificateAuthorityData []byte `json:"certificate-authority-data,omitempty"`

	// MonitorHelmReleases to trim down the number of revisions per release
	MonitorHelmReleases bool
}

// ToClusterType generates an external types.Cluster to be shared over REST
func (c *Cluster) ToClusterType() *types.Cluster {
	serv := types.Kube

	if c.AWSIntegrationID != 0 {
		serv = types.EKS
	} else if c.GCPIntegrationID != 0 {
		serv = types.GKE
	} else if c.DOIntegrationID != 0 {
		serv = types.DOKS
	} else if c.AzureIntegrationID != 0 {
		serv = types.AKS
	}

	return &types.Cluster{
		ID:                                c.ID,
		ProjectID:                         c.ProjectID,
		Name:                              c.Name,
		VanityName:                        c.VanityName,
		Server:                            c.Server,
		Service:                           serv,
		AgentIntegrationEnabled:           c.AgentIntegrationEnabled,
		InfraID:                           c.InfraID,
		AWSIntegrationID:                  c.AWSIntegrationID,
		AWSClusterID:                      c.AWSClusterID,
		PreviewEnvsEnabled:                c.PreviewEnvsEnabled,
		Status:                            c.Status,
		ProvisionedBy:                     c.ProvisionedBy,
		CloudProvider:                     c.CloudProvider,
		CloudProviderCredentialIdentifier: c.CloudProviderCredentialIdentifier,
	}
}

// ClusterCandidate is a cluster integration that requires additional action
// from the user to set up.
type ClusterCandidate struct {
	gorm.Model

	// The auth mechanism that this candidate will parse for
	AuthMechanism ClusterAuth `json:"auth_mechanism"`

	// The project that this integration belongs to
	ProjectID uint `json:"project_id"`

	// CreatedClusterID is the ID of the cluster that's eventually
	// created
	CreatedClusterID uint `json:"created_cluster_id"`

	// Resolvers are the list of resolvers: once all resolvers are "resolved," the
	// cluster will be created
	Resolvers []ClusterResolver `json:"resolvers"`

	// Name of the cluster
	Name string `json:"name"`

	// Server endpoint for the cluster
	Server string `json:"server"`

	// Name of the context that this was created from, if it exists
	ContextName string `json:"context_name"`

	// ------------------------------------------------------------------
	// All fields below this line are encrypted before storage
	// ------------------------------------------------------------------

	// The best-guess for the AWSClusterID, which is required by aws auth mechanisms
	// See https://github.com/kubernetes-sigs/aws-iam-authenticator#what-is-a-cluster-id
	AWSClusterIDGuess []byte `json:"aws_cluster_id_guess"`

	// The raw kubeconfig
	Kubeconfig []byte `json:"kubeconfig"`
}

func (cc *ClusterCandidate) ToClusterCandidateType() *types.ClusterCandidate {
	resolvers := make([]types.ClusterResolver, 0)

	for _, resolver := range cc.Resolvers {
		resolvers = append(resolvers, *resolver.ToClusterResolverType())
	}

	return &types.ClusterCandidate{
		ID:                cc.ID,
		ProjectID:         cc.ProjectID,
		CreatedClusterID:  cc.CreatedClusterID,
		Name:              cc.Name,
		Server:            cc.Server,
		ContextName:       cc.ContextName,
		Resolvers:         resolvers,
		AWSClusterIDGuess: string(cc.AWSClusterIDGuess),
	}
}

// ClusterResolver is an action that must be resolved to set up
// a Cluster
type ClusterResolver struct {
	gorm.Model

	// The ClusterCandidate that this is resolving
	ClusterCandidateID uint `json:"cluster_candidate_id"`

	// One of the ClusterResolverNames
	Name types.ClusterResolverName `json:"name"`

	// Resolved is true if this has been resolved, false otherwise
	Resolved bool `json:"resolved"`

	// Data is additional data for resolving the action, for example a file name,
	// context name, etc
	Data []byte `json:"data,omitempty"`
}

func (cr *ClusterResolver) ToClusterResolverType() *types.ClusterResolver {
	info := types.ClusterResolverInfos[cr.Name]

	data := make(types.ClusterResolverData)

	json.Unmarshal(cr.Data, &data)

	return &types.ClusterResolver{
		ID:                 cr.ID,
		ClusterCandidateID: cr.ClusterCandidateID,
		Name:               cr.Name,
		Resolved:           cr.Resolved,
		Docs:               info.Docs,
		Fields:             info.Fields,
		Data:               data,
	}
}
