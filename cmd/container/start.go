package container

import (
	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/spf13/cobra"
)

type startOptions struct {
	attach        bool
	openStdin     bool
	detachKeys    string
	checkpoint    string
	checkpointDir string

	containers []string
}

func NewStartCommand(sudockerCli *cmd.SudockerCli) *cobra.Command {
	var opts startOptions

	cmd := &cobra.Command{
		Use:   "start [OPTIONS] CONTAINER [CONTAINER...]",
		Short: "Start one or more stopped containers",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containers = args
			return nil
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.attach, "attach", "a", false, "Attach STDOUT/STDERR and forward signals")
	flags.BoolVarP(&opts.openStdin, "interactive", "i", false, "Attach container's STDIN")
	flags.StringVar(&opts.detachKeys, "detach-keys", "", "Override the key sequence for detaching a container")

	flags.StringVar(&opts.checkpoint, "checkpoint", "", "Restore from this checkpoint")
	flags.SetAnnotation("checkpoint", "experimental", nil)
	flags.StringVar(&opts.checkpointDir, "checkpoint-dir", "", "Use a custom checkpoint storage directory")
	flags.SetAnnotation("checkpoint-dir", "experimental", nil)
	return cmd
}
