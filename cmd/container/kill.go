package container

import (
	"github.com/DeJeune/sudocker/cli"
	"github.com/spf13/cobra"
)

type killOptions struct {
	signal string

	containers []string
}

// NewKillCommand creates a new cobra.Command for `docker kill`
func NewKillCommand() *cobra.Command {
	var opts killOptions

	cmd := &cobra.Command{
		Use:   "kill [OPTIONS] CONTAINER [CONTAINER...]",
		Short: "Kill one or more running containers",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.signal, "signal", "s", "KILL", "Signal to send to the container")
	return cmd
}
