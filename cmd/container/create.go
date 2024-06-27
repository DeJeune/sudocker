package container

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/DeJeune/sudocker/runtime/pkg/cgroups/fs"
	"github.com/DeJeune/sudocker/runtime/pkg/container"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
			return runCreate(cmd.Context(), sudockerCli, cmd.Flags(), &options, copts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&options.name, "name", "", "Assign a name to the container")
	flags.StringVar(&options.pull, "pull", PullImageMissing, `Pull image before creating ("`+PullImageAlways+`", "|`+PullImageMissing+`", "`+PullImageNever+`")`)
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Suppress the pull output")
	copts = addFlags(flags)
	return cmd
}

func runCreate(ctx context.Context, sudockerCli cmd.Cli, flags *pflag.FlagSet, options *createOptions, copts *containerOptions) error {
	containerConfig, err := parse(flags, copts)
	if err != nil {
		reportError(sudockerCli.Err(), "create", err.Error(), true)
		return cli.StatusError{StatusCode: 125}
	}
	id, err := newContainer(ctx, sudockerCli, containerConfig, options)
	if err != nil {
		return nil
	}
	_, _ = fmt.Fprintln(sudockerCli.Out(), id)
	return nil
}

func newContainer(ctx context.Context, sudockerCli cmd.Cli, containerConfig *containerConfig, options *createOptions) (containerId string, err error) {
	containerId = container.GenerateContainerID()
	cg := containerConfig.Config
	hostConfig := containerConfig.HostConfig
	parent, writePipe := container.NewParentProcess(ctx, sudockerCli, cg, hostConfig, containerId)
	if err := parent.Start(); err != nil {
		return containerId, errors.Errorf("Failed to start parent process: %v", err)
	}

	if err := container.RecordContainerInfo(parent.Process.Pid, cg.Cmd, options.name, containerId); err != nil {
		return containerId, err
	}

	cgroupConfig := &config.Cgroup{
		Name:      "sudocker-cgroup",
		Rootless:  false,
		Resources: hostConfig.Resources,
	}

	cgroupManager, err := fs.NewManager(cgroupConfig, nil)
	defer cgroupManager.Destroy()
	if err != nil {
		return containerId, err
	}
	err = cgroupManager.Set(cgroupConfig.Resources)
	if err != nil {
		return containerId, err
	}
	err = cgroupManager.Apply(parent.Process.Pid)
	if err != nil {
		return containerId, err
	}
	sendInitCommand(cg.Cmd, writePipe)
	if cg.Tty {
		if err := parent.Wait(); err != nil {
			return containerId, errors.Errorf("Parent process failed: %v", err)
		}
		if err := container.DeleteVolumes("/root/", hostConfig.Binds); err != nil {
			return containerId, errors.Errorf("Umount volumes failed: %v", err)
		}
		if err := container.DeleteContainerInfo(containerId); err != nil {
			return containerId, errors.Errorf("delete container info failed: %v", err)
		}
	}
	return containerId, nil
}

// sendInitCommand 通过writePipe将指令发送给子进程
func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	_, _ = writePipe.WriteString(command)
	_ = writePipe.Close()
}
