package network

import (
	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/spf13/cobra"
)

func NewNetworkCommand(sudockerCli *cmd.SudockerCli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Manage networks",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.HelpFunc()(cmd, args)
			return nil
		},
	}
	cmd.AddCommand(
		NewCreateCommand(sudockerCli),
		NewListCommand(sudockerCli),
		NewRemoveCommand(sudockerCli),
	)
	return cmd
}
