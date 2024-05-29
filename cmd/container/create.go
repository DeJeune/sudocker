package container

import (
	"context"
	"fmt"

	"github.com/DeJeune/sudocker/cmd"
	"github.com/spf13/cobra"
)

const (
	PullImageAlways  = "always"
	PullImageMissing = "missing" // Default (matches previous behavior)
	PullImageNever   = "never"
)

type createOptions struct {
	name      string
	platform  string
	untrusted bool
	pull      string // always, missing, never
	quiet     bool
}

func NewCreateCommand(sudockerCli *cmd.SudockerCli) *cobra.Command {
	var options createOptions
	var copts *containerOptions
	cmd := &cobra.Command{
		Use:   "create [OPTIONS] IMAGE [COMMAND] [ARG...]",
		Short: "Create a new container",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("create called")
			copts.Image = args[0]
			if len(args) > 1 {
				copts.Args = args[1:]
			}
			containerConfig, err := parse(cmd.Flags(), copts, "Linux")
			if err != nil {
				return err
			}
			return newContainer(cmd.Context(), sudockerCli, containerConfig, &options)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&options.name, "name", "", "Assign a name to the container")
	flags.StringVar(&options.pull, "pull", PullImageMissing, `Pull image before creating ("`+PullImageAlways+`", "|`+PullImageMissing+`", "`+PullImageNever+`")`)
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Suppress the pull output")

	return cmd
}

func newContainer(ctx context.Context, sudockerCli *cmd.SudockerCli, containerConfig *containerConfig, options *createOptions) (*Container, error) {

}
