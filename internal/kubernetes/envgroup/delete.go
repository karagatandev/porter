package envgroup

import "github.com/karagatandev/porter/internal/kubernetes"

func DeleteEnvGroup(agent *kubernetes.Agent, name, namespace string) error {
	if err := agent.DeleteVersionedSecret(name, namespace); err != nil {
		return err
	}

	if err := agent.DeleteVersionedConfigMap(name, namespace); err != nil {
		return err
	}

	return nil
}
