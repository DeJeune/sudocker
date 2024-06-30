package network

import (
	"context"

	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cli/opts"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/runtime/pkg/network"
	"github.com/spf13/cobra"
)

type listOptions struct {
	quiet   bool
	noTrunc bool
	format  string
	filter  opts.FilterOpt
}

func NewListCommand(sudockerCli cmd.Cli) *cobra.Command {
	options := &listOptions{
		filter: opts.NewFilterOpt(),
	}

	cmd := &cobra.Command{
		Use:     "ls [OPTIONS] NETWORK",
		Aliases: []string{"list"},
		Short:   "List networks",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd.Context(), sudockerCli, options)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display network IDs")
	flags.BoolVar(&options.noTrunc, "no-trunc", false, "Do not truncate the output")
	flags.VarP(&options.filter, "filter", "f", `Provide filter values (e.g. "driver=bridge")`)
	return cmd
}

func runList(ctx context.Context, sudockerCli cmd.Cli, options *listOptions) error {
	return network.ListNetwork()
}
