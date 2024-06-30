package container

import (
	"context"
	"fmt"
	"strings"

	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/runtime/pkg/container"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type stopOptions struct {
	timeout        int
	timeChanged    bool
	timeoutChanged bool
	containers     []string
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
			return runStop(cmd.Context(), sudockerCli, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container stop, docker stop",
		},
	}

	flags := cmd.Flags()
	flags.IntVarP(&opts.timeout, "time", "t", 10, "Seconds to wait for stop before killing it")
	return cmd
}

func runStop(ctx context.Context, sudockerCli cmd.Cli, opts *stopOptions) error {
	errChan := parallelOperation(ctx, opts.containers, func(ctx context.Context, containerId string) error {

		return container.StopContainer(containerId)
	})
	var errs []string
	for _, ctr := range opts.containers {
		if err := <-errChan; err != nil {
			errs = append(errs, err.Error())
			continue
		}
		_, _ = fmt.Fprintln(sudockerCli.Out(), ctr)
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
