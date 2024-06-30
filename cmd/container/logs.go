package container

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/runtime/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type logsOptions struct {
	follow     bool
	since      string
	until      string
	timestamps bool
	details    bool
	tail       string

	containerId string
}

func NewLogsCommand(sudockerCli *cmd.SudockerCli) *cobra.Command {
	var opts logsOptions

	cmd := &cobra.Command{
		Use:   "logs [OPTIONS] CONTAINER",
		Short: "Fetch the logs of a container",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.containerId = args[0]
			return runLogs(cmd.Context(), sudockerCli, &opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container logs, docker logs",
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.follow, "follow", "f", false, "Follow log output")
	flags.StringVar(&opts.since, "since", "", `Show logs since timestamp (e.g. "2013-01-02T13:23:37Z") or relative (e.g. "42m" for 42 minutes)`)
	flags.StringVar(&opts.until, "until", "", `Show logs before a timestamp (e.g. "2013-01-02T13:23:37Z") or relative (e.g. "42m" for 42 minutes)`)
	flags.SetAnnotation("until", "version", []string{"1.35"})
	flags.BoolVarP(&opts.timestamps, "timestamps", "t", false, "Show timestamps")
	flags.BoolVar(&opts.details, "details", false, "Show extra details provided to logs")
	flags.StringVarP(&opts.tail, "tail", "n", "all", "Number of lines to show from the end of the logs")
	return cmd
}

func runLogs(ctx context.Context, sudockerCli *cmd.SudockerCli, opts *logsOptions) error {
	logFileLocation := fmt.Sprintf(utils.InfoLocFormat, opts.containerId) + utils.GetLogfile(opts.containerId)
	file, err := os.Open(logFileLocation)
	if err != nil {
		return errors.Errorf("Log container open file %s error %v", logFileLocation, err)

	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return errors.Errorf("Log container read file %s error %v", logFileLocation, err)
	}
	_, err = fmt.Fprint(os.Stdout, string(content))
	if err != nil {
		return errors.Errorf("Log container Fprint  error %v", err)
	}
	// if c.Config.Tty {
	// 	_, err = io.Copy(dockerCli.Out(), responseBody)
	// } else {
	// 	_, err = stdcopy.StdCopy(dockerCli.Out(), dockerCli.Err(), responseBody)
	// }
	return nil
}
