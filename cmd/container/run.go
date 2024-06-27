package container

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"syscall"

	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type runOptions struct {
	createOptions
	detach     bool
	sigProxy   bool
	detachKeys string
}

func NewRunCommand(sudockerCli *cmd.SudockerCli) *cobra.Command {
	var copts *containerOptions
	var options runOptions

	runCmd := &cobra.Command{
		Use:   "run [OPTIONS] IMAGE [COMMAND] [ARG...]",
		Short: "Create and Run a container from a image",
		Long:  `Create and Run a container from a image`,
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("run called")
			copts.Image = args[0]
			if len(args) > 1 {
				copts.Args = args[1:]
			}
			return runRun(cmd.Context(), sudockerCli, cmd.Flags(), &options, copts)
		},
		Annotations: map[string]string{
			"category-top": "1",
			"aliases":      "sudocker container run, sudocker run",
		},
	}
	flags := runCmd.Flags()
	// Here you will define your flags and configuration settings.
	flags.BoolVarP(&options.detach, "detach", "d", true, "Run the container in the background")
	flags.StringVarP(&options.name, "name", "n", "", "Assign a name to the container")
	copts = addFlags(flags)
	return runCmd
}

func runRun(ctx context.Context, sudockerCli cmd.Cli, flags *pflag.FlagSet, ropts *runOptions, copts *containerOptions) error {
	containerConfig, err := parse(flags, copts)
	if err != nil {
		reportError(sudockerCli.Err(), "run", err.Error(), true)
		return cli.StatusError{StatusCode: 125}
	}
	return runContainer(ctx, sudockerCli, ropts, copts, containerConfig)
}

func runContainer(ctx context.Context, sudockerCli cmd.Cli, runOpts *runOptions, copts *containerOptions, containerCfg *containerConfig) error {
	config := containerCfg.Config
	stdout, stderr := sudockerCli.Out(), sudockerCli.Err()

	if !runOpts.detach {
		if err := sudockerCli.In().CheckTty(config.AttachStdin, config.Tty); err != nil {
			return err
		}
	} else {
		if copts.attach.Len() != 0 {
			return errors.New("Conflicting options: -a and -d")
		}

		config.AttachStdin = false
		config.AttachStdout = false
		config.AttachStderr = false
		config.StdinOnce = false
	}
	ctx, cancelFun := context.WithCancel(ctx)
	defer cancelFun()
	containerID, err := newContainer(ctx, sudockerCli, containerCfg, &runOpts.createOptions)
	if err != nil {
		reportError(stderr, "run", err.Error(), true)
		return runStartContainerErr(err)
	}

	var (
		waitDisplayID chan struct{}
	)
	if !config.AttachStdout && !config.AttachStderr {
		// Make this asynchronous to allow the client to write to stdin before having to read the ID
		waitDisplayID = make(chan struct{})
		go func() {
			defer close(waitDisplayID)
			_, _ = fmt.Fprintln(stdout, containerID)
		}()
	}

	// attach := config.AttachStdin || config.AttachStdout || config.AttachStderr

	// if attach {
	// 	detachKeys := sudockerCli.ConfigFile().DetachKeys
	// 	if runOpts.detachKeys != "" {
	// 		detachKeys = runOpts.detachKeys
	// 	}

	// 	closeFn, err := attachContainer(ctx, sudockerCli, containerID, &errCh, config, container.AttachOptions{
	// 		Stream:     true,
	// 		Stdin:      config.AttachStdin,
	// 		Stdout:     config.AttachStdout,
	// 		Stderr:     config.AttachStderr,
	// 		DetachKeys: detachKeys,
	// 	})
	// 	if err != nil {
	// 		return err
	// 	}
	// 	defer closeFn()
	// }

	// var (
	// 	waitDisplayID chan struct{}
	// 	errCh         chan error
	// )
	// if !config.AttachStdout && !config.AttachStderr {
	// 	// Make this asynchronous to allow the client to write to stdin before having to read the ID
	// 	waitDisplayID = make(chan struct{})
	// 	go func() {
	// 		defer close(waitDisplayID)
	// 		_, _ = fmt.Fprintln(stdout, containerID)
	// 	}()
	// }

	if (config.AttachStdin || config.AttachStdout || config.AttachStderr) && config.Tty && sudockerCli.Out().IsTerminal() {
		if err := MonitorTtySize(ctx, sudockerCli, containerID, false); err != nil {
			_, _ = fmt.Fprintln(stderr, "Error monitoring TTY size:", err)
		}
	}

	return nil
}

func reportError(stderr io.Writer, name string, str string, withHelp bool) {
	str = strings.TrimSuffix(str, ".") + "."
	if withHelp {
		str += "\nSee 'sudocker " + name + " --help'."
	}
	_, _ = fmt.Fprintln(stderr, "sudocker:", str)
}

// 如果容器启动失败并显示“not found”/“no such”错误，返回127；如果容器启动失败并显示“权限被拒绝”错误，返回126；对于一般的Docker守护进程故障，返回125。
func runStartContainerErr(err error) error {
	trimmedErr := strings.TrimPrefix(err.Error(), "Error response from daemon: ")
	statusError := cli.StatusError{StatusCode: 125}
	if strings.Contains(trimmedErr, "executable file not found") ||
		strings.Contains(trimmedErr, "no such file or directory") ||
		strings.Contains(trimmedErr, "system cannot find the file specified") {
		statusError = cli.StatusError{StatusCode: 127}
	} else if strings.Contains(trimmedErr, syscall.EACCES.Error()) ||
		strings.Contains(trimmedErr, syscall.EISDIR.Error()) {
		statusError = cli.StatusError{StatusCode: 126}
	}

	return statusError
}
