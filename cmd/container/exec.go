package container

import (
	"fmt"

	"github.com/DeJeune/sudocker/cli"
	"github.com/spf13/cobra"
)

type execOptions struct {
	detachKeys  string
	interactive bool
	tty         bool
	detach      bool
	user        string
	privileged  bool
}

func NewExecCommand() *cobra.Command {
	var options execOptions

	cmd := &cobra.Command{
		Use:   "exec [OPTIONS] CONTAINER COMMAND [ARG...]",
		Short: "Run a command in a running container",
		Args:  cli.RequiresMinArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("exec the contaiener")
			return nil
		},
	}

	flags := cmd.Flags()
	flags.SetInterspersed(false)

	flags.StringVarP(&options.detachKeys, "detach-keys", "", "", "Override the key sequence for detaching a container")
	flags.BoolVarP(&options.interactive, "interactive", "i", false, "Keep STDIN open even if not attached")
	flags.BoolVarP(&options.tty, "tty", "t", false, "Allocate a pseudo-TTY")
	flags.BoolVarP(&options.detach, "detach", "d", false, "Detached mode: run command in the background")
	flags.StringVarP(&options.user, "user", "u", "", "Username or UID (format: <name|uid>[:<group|gid>])")
	flags.BoolVarP(&options.privileged, "privileged", "", false, "Give extended privileges to the command")

	return cmd
}
