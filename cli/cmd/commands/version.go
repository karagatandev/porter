package commands

import (
	"fmt"

	"github.com/karagatandev/porter/cli/cmd/config"
	"github.com/spf13/cobra"
)

func registerCommand_Version(_ config.CLIConfig) *cobra.Command {
	versionCmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"v", "--version"},
		Short:   "Prints the version of the Porter CLI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(config.Version)
		},
	}
	return versionCmd
}
