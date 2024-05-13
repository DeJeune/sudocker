package container

import (
	"github.com/DeJeune/sudocker/cli"
	"github.com/spf13/cobra"
)

type restartOptions struct {
	nSeconds        int
	nSecondsChanged bool

	containers []string
}

func NewRestartCommand() *cobra.Command {
	var opts restartOptions

	cmd := &cobra.Command{
		Use:   "restart [OPTIONS] CONTAINER [CONTAINER...]",
		Short: "Restart one or more containers",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			opts.nSecondsChanged = cmd.Flags().Changed("time")
			return nil
		},
	}

	flags := cmd.Flags()
	flags.IntVarP(&opts.nSeconds, "time", "t", 10, "Seconds to wait for stop before killing the container")
	return cmd
}
