package network

import (
	"context"

	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/runtime/pkg/network"
	"github.com/spf13/cobra"
)

type removeOptions struct {
	force bool
}

func NewRemoveCommand(sudockerCli cmd.Cli) *cobra.Command {
	opts := &removeOptions{}

	cmd := &cobra.Command{
		Use:     "rm NETWORK [NETWORK...]",
		Aliases: []string{"remove"},
		Short:   "Remove one or more networks",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd.Context(), sudockerCli, args, opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.force, "force", "f", false, "Do not error if the network does not exist")
	return cmd
}

func runRemove(ctx context.Context, sudockerCli cmd.Cli, networks []string, opts *removeOptions) error {
	for _, name := range networks {
		if err := network.DeleteNetwork(name); err != nil {
			return err
		}
	}
	return nil
}
