package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/google/uuid"
	api "github.com/karagatandev/porter/api/client"
	"github.com/karagatandev/porter/api/types"
	"github.com/karagatandev/porter/cli/cmd/config"
	"github.com/spf13/cobra"
)

func registerCommand_Target(cliConf config.CLIConfig) *cobra.Command {
	targetCmd := &cobra.Command{
		Use:     "target",
		Aliases: []string{"targets"},
		Short:   "Commands that control Porter target settings",
	}

	createTargetCmd := &cobra.Command{
		Use:   "create --name [name]",
		Short: "Creates a deployment target",
		Run: func(cmd *cobra.Command, args []string) {
			err := checkLoginAndRunWithConfig(cmd, cliConf, args, createTarget)
			if err != nil {
				os.Exit(1)
			}
		},
	}

	var targetName string
	createTargetCmd.Flags().StringVar(&targetName, "name", "", "Name of deployment target")
	targetCmd.AddCommand(createTargetCmd)

	listTargetCmd := &cobra.Command{
		Use:   "list",
		Short: "Lists the deployment targets for the logged in user",
		Long: `Lists the deployment targets in the project

The following columns are returned:
* ID:          id of the deployment target
* NAME:        name of the deployment target
* CLUSTER-ID:  id of the cluster associated with the deployment target
* DEFAULT:     whether the deployment target is the default target for the cluster

If the --preview flag is set, only deployment targets for preview environments will be returned.
`,
		Run: func(cmd *cobra.Command, args []string) {
			err := checkLoginAndRunWithConfig(cmd, cliConf, args, listTargets)
			if err != nil {
				os.Exit(1)
			}
		},
	}

	var includePreviews bool
	listTargetCmd.Flags().BoolVar(&includePreviews, "preview", false, "List preview environments")
	targetCmd.AddCommand(listTargetCmd)

	deleteTargetCmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes a deployment target",
		Long:  `Deletes a deployment target in the project. Currently, this command only supports the deletion of preview environments.`,
		Run: func(cmd *cobra.Command, args []string) {
			err := checkLoginAndRunWithConfig(cmd, cliConf, args, deleteTarget)
			if err != nil {
				os.Exit(1)
			}
		},
	}

	deleteTargetCmd.Flags().StringVar(&targetName, "name", "", "Name of deployment target")
	deleteTargetCmd.Flags().BoolP("force", "f", false, "Force deletion without confirmation")
	deleteTargetCmd.MarkFlagRequired("name") // nolint:errcheck,gosec
	targetCmd.AddCommand(deleteTargetCmd)

	return targetCmd
}

func createTarget(ctx context.Context, _ *types.GetAuthenticatedUserResponse, client api.Client, cliConf config.CLIConfig, featureFlags config.FeatureFlags, cmd *cobra.Command, args []string) error {
	targetName, err := cmd.Flags().GetString("name")
	if err != nil {
		return fmt.Errorf("error finding name flag: %w", err)
	}

	resp, err := client.CreateDeploymentTarget(ctx, cliConf.Project, &types.CreateDeploymentTargetRequest{
		Name:      targetName,
		ClusterId: cliConf.Cluster,
	})
	if err != nil {
		return err
	}

	_, _ = color.New(color.FgGreen).Printf("Created target with name %s and id %s\n", targetName, resp.DeploymentTargetID)

	return nil
}

func listTargets(ctx context.Context, user *types.GetAuthenticatedUserResponse, client api.Client, cliConf config.CLIConfig, featureFlags config.FeatureFlags, cmd *cobra.Command, args []string) error {
	includePreviews, err := cmd.Flags().GetBool("preview")
	if err != nil {
		return fmt.Errorf("error finding preview flag: %w", err)
	}

	resp, err := client.ListDeploymentTargets(ctx, cliConf.Project, includePreviews)
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}

	targets := *resp

	sort.Slice(targets.DeploymentTargets, func(i, j int) bool {
		if targets.DeploymentTargets[i].ClusterID != targets.DeploymentTargets[j].ClusterID {
			return targets.DeploymentTargets[i].ClusterID < targets.DeploymentTargets[j].ClusterID
		}
		return targets.DeploymentTargets[i].Name < targets.DeploymentTargets[j].Name
	})

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 3, 8, 0, '\t', tabwriter.AlignRight)

	if includePreviews {
		fmt.Fprintf(w, "%s\t%s\t%s\n", "ID", "NAME", "CLUSTER-ID")
		for _, target := range targets.DeploymentTargets {
			fmt.Fprintf(w, "%s\t%s\t%d\n", target.ID, target.Name, target.ClusterID)
		}
	} else {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "ID", "NAME", "CLUSTER-ID", "DEFAULT")
		for _, target := range targets.DeploymentTargets {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", target.ID, target.Name, target.ClusterID, checkmark(target.IsDefault))
		}
	}

	_ = w.Flush()

	return nil
}

func deleteTarget(ctx context.Context, _ *types.GetAuthenticatedUserResponse, client api.Client, cliConf config.CLIConfig, featureFlags config.FeatureFlags, cmd *cobra.Command, args []string) error {
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return fmt.Errorf("error finding name flag: %w", err)
	}
	if name == "" {
		return fmt.Errorf("name flag must be set")
	}

	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return fmt.Errorf("error finding force flag: %w", err)
	}

	var confirmed bool
	if !force {
		confirmed, err = confirmAction(fmt.Sprintf("Are you sure you want to delete target '%s'?", name))
		if err != nil {
			return fmt.Errorf("error confirming action: %w", err)
		}
	}
	if !confirmed && !force {
		color.New(color.FgYellow).Println("Deletion aborted") // nolint:errcheck,gosec
		return nil
	}

	// assume deletion will be for preview environments only for now
	dts, err := client.ListDeploymentTargets(ctx, cliConf.Project, true)
	if err != nil {
		return fmt.Errorf("error listing targets: %w", err)
	}

	var targetID uuid.UUID
	for _, dt := range dts.DeploymentTargets {
		if dt.Name == name {
			targetID = dt.ID
			break
		}
	}
	if targetID == uuid.Nil {
		return fmt.Errorf("target '%s' not found", name)
	}

	err = client.DeleteDeploymentTarget(ctx, cliConf.Project, targetID)
	if err != nil {
		return fmt.Errorf("error deleting target: %w", err)
	}

	color.New(color.FgGreen).Printf("Deleted target '%s'\n", name) // nolint:errcheck,gosec

	return nil
}

func confirmAction(prompt string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [Y/n]: ", prompt)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("error reading input: %w", err)
	}

	response = strings.TrimSpace(response)
	confirmed := strings.ToLower(response) == "y" || response == ""

	return confirmed, nil
}

func checkmark(b bool) string {
	if b {
		return "✓"
	}

	return ""
}
