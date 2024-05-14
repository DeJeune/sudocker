package container

import (
	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/spf13/cobra"
)

type stopOptions struct {
	time        int
	timeChanged bool

	containers []string
}

func NewStopCommand(sudockerCli *cmd.SudockerCli) *cobra.Command {
	var opts stopOptions

	cmd := &cobra.Command{
		Use:   "stop [OPTIONS] CONTAINER [CONTAINER...]",
		Short: "Stop one or more running containers",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			opts.timeChanged = cmd.Flags().Changed("time")
			return nil
		},
	}

	flags := cmd.Flags()
	flags.IntVarP(&opts.time, "time", "t", 10, "Seconds to wait for stop before killing it")
	return cmd
}
