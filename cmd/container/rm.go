package container

import (
	"context"
	"fmt"
	"strings"

	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/runtime/pkg/container"
	"github.com/docker/docker/errdefs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type rmOptions struct {
	rmVolumes bool
	rmLink    bool
	force     bool

	containers []string
}

func NewRmCommand(sudockerCli *cmd.SudockerCli) *cobra.Command {

	var opts rmOptions

	cmd := &cobra.Command{
		Use:     "rm [OPTIONS] CONTAINER [CONTAINER...]",
		Short:   "Remove one or more containers",
		Aliases: []string{"remove"},
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("rm called")
			opts.containers = args
			return runRm(cmd.Context(), sudockerCli, &opts)
		},
		Annotations: map[string]string{
			"aliases": "sudocker container rm, sudocker container remove, sudocker rm",
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&opts.rmVolumes, "volumes", "v", false, "Remove the volumes associated with the container")
	flags.BoolVarP(&opts.rmLink, "link", "l", false, "Remove the specified link")
	flags.BoolVarP(&opts.force, "force", "f", false, "Force the removal of a running container (uses SIGKILL)")

	return cmd
}

func runRm(ctx context.Context, sudockerCli cmd.Cli, opts *rmOptions) error {
	var errs []string
	errChan := parallelOperation(ctx, opts.containers, func(ctx context.Context, containerId string) error {
		return container.RmContainer(containerId, container.RemoveOptions{
			Force:         opts.force,
			RemoveVolumes: opts.rmVolumes,
			RemoveLinks:   opts.rmLink,
		})
	})

	for _, name := range opts.containers {
		if err := <-errChan; err != nil {
			if opts.force && errdefs.IsNotFound(err) {
				fmt.Fprintln(sudockerCli.Err(), err)
				continue
			}
			errs = append(errs, err.Error())
			continue
		}
		fmt.Fprintln(sudockerCli.Out(), name)
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
