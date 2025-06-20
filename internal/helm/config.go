package helm

import (
	"context"
	"errors"
	"io/ioutil"
	"time"

	"github.com/karagatandev/porter/internal/kubernetes"
	"github.com/karagatandev/porter/internal/models"
	"github.com/karagatandev/porter/internal/repository"
	"github.com/karagatandev/porter/pkg/logger"
	"github.com/stefanmcshane/helm/pkg/action"
	"github.com/stefanmcshane/helm/pkg/chartutil"
	"github.com/stefanmcshane/helm/pkg/kube"
	kubefake "github.com/stefanmcshane/helm/pkg/kube/fake"
	"github.com/stefanmcshane/helm/pkg/storage"
	"golang.org/x/oauth2"
	k8s "k8s.io/client-go/kubernetes"
)

// Form represents the options for connecting to a cluster and
// creating a Helm agent
type Form struct {
	Cluster                   *models.Cluster `form:"required"`
	Repo                      repository.Repository
	DigitalOceanOAuth         *oauth2.Config
	Storage                   string `json:"storage" form:"oneof=secret configmap memory" default:"secret"`
	Namespace                 string `json:"namespace"`
	AllowInClusterConnections bool
	Timeout                   time.Duration // optional
}

// GetAgentOutOfClusterConfig creates a new Agent from outside the cluster using
// the underlying kubernetes.GetAgentOutOfClusterConfig method
func GetAgentOutOfClusterConfig(ctx context.Context, form *Form, l *logger.Logger) (*Agent, error) {
	// create a kubernetes agent
	conf := &kubernetes.OutOfClusterConfig{
		Cluster:                   form.Cluster,
		DefaultNamespace:          form.Namespace,
		Repo:                      form.Repo,
		DigitalOceanOAuth:         form.DigitalOceanOAuth,
		AllowInClusterConnections: form.AllowInClusterConnections,
		Timeout:                   form.Timeout,
	}

	k8sAgent, err := kubernetes.GetAgentOutOfClusterConfig(ctx, conf)
	if err != nil {
		return nil, err
	}

	return GetAgentFromK8sAgent(form.Storage, form.Namespace, l, k8sAgent)
}

// GetAgentFromK8sAgent creates a new Agent
func GetAgentFromK8sAgent(stg string, ns string, l *logger.Logger, k8sAgent *kubernetes.Agent) (*Agent, error) {
	// clientset, ok := k8sAgent.Clientset.(*k8s.Clientset)

	// if !ok {
	// 	return nil, errors.New("Agent Clientset was not of type *(k8s.io/client-go/kubernetes).Clientset")
	// }

	// actionConf := &action.Configuration{
	// 	RESTClientGetter: k8sAgent.RESTClientGetter,
	// 	KubeClient:       kube.New(k8sAgent.RESTClientGetter),
	// 	Releases:         StorageMap[stg](l, clientset.CoreV1(), ns),
	// 	Log:              l.Printf,
	// }

	actionConf := &action.Configuration{}

	if err := actionConf.Init(k8sAgent.RESTClientGetter, ns, stg, l.Printf); err != nil {
		return nil, err
	}

	// use k8s agent to create Helm agent
	return &Agent{
		ActionConfig: actionConf,
		K8sAgent:     k8sAgent,
		namespace:    ns,
	}, nil
}

// GetAgentInClusterConfig creates a new Agent from inside the cluster using
// the underlying kubernetes.GetAgentInClusterConfig method
func GetAgentInClusterConfig(ctx context.Context, form *Form, l *logger.Logger) (*Agent, error) {
	// create a kubernetes agent
	k8sAgent, err := kubernetes.GetAgentInClusterConfig(ctx, form.Namespace)
	if err != nil {
		return nil, err
	}

	clientset, ok := k8sAgent.Clientset.(*k8s.Clientset)

	if !ok {
		return nil, errors.New("Agent Clientset was not of type *(k8s.io/client-go/kubernetes).Clientset")
	}

	// use k8s agent to create Helm agent
	return &Agent{
		ActionConfig: &action.Configuration{
			RESTClientGetter: k8sAgent.RESTClientGetter,
			KubeClient:       kube.New(k8sAgent.RESTClientGetter),
			Releases:         StorageMap[form.Storage](l, clientset.CoreV1(), form.Namespace),
			Log:              l.Printf,
		},
		K8sAgent: k8sAgent,
	}, nil
}

// GetAgentTesting creates a new Agent using an optional existing storage class
func GetAgentTesting(form *Form, storage *storage.Storage, l *logger.Logger, k8sAgent *kubernetes.Agent) *Agent {
	testStorage := storage

	if testStorage == nil {
		testStorage = StorageMap["memory"](nil, nil, "")
	}

	return &Agent{
		ActionConfig: &action.Configuration{
			Releases: testStorage,
			KubeClient: &kubefake.FailingKubeClient{
				PrintingKubeClient: kubefake.PrintingKubeClient{
					Out: ioutil.Discard,
				},
			},
			Capabilities: chartutil.DefaultCapabilities,
			Log:          l.Printf,
		},
		K8sAgent: k8sAgent,
	}
}
