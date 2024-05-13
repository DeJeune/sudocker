package container

import (
	"fmt"

	"github.com/DeJeune/sudocker/cli"
	"github.com/spf13/cobra"
)

type rmOptions struct {
	rmVolumes bool
	rmLink    bool
	force     bool

	containers []string
}

func NewRmCommand() *cobra.Command {

	var opts rmOptions

	cmd := &cobra.Command{
		Use:     "sudocker container rm [OPTIONS] CONTAINER [CONTAINER...]",
		Short:   "Remove one or more containers",
		Aliases: []string{"sudocker container run"},
		Args:    cli.RequiresMinArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("rm called")
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&opts.rmVolumes, "volumes", "v", false, "Remove the volumes associated with the container")
	flags.BoolVarP(&opts.rmLink, "link", "l", false, "Remove the specified link")
	flags.BoolVarP(&opts.force, "force", "f", false, "Force the removal of a running container (uses SIGKILL)")

	return cmd
}
