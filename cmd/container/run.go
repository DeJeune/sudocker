package container

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"syscall"

	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/runtime/pkg/container"
	"github.com/DeJeune/sudocker/runtime/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	flags.BoolVarP(&options.detach, "detach", "d", false, "Run the container in the background")
	flags.StringVarP(&options.name, "name", "n", "", "Assign a name to the container")
	copts = addFlags(flags)
	runCmd.RegisterFlagCompletionFunc(
		"env",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return os.Environ(), cobra.ShellCompDirectiveNoFileComp
		},
	)
	runCmd.RegisterFlagCompletionFunc(
		"env-file",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveDefault
		},
	)
	return runCmd
}

func runRun(ctx context.Context, sudockerCli *cmd.SudockerCli, flags *pflag.FlagSet, ropts *runOptions, copts *containerOptions) error {
	// newEnv := []string{}
	// copts.env = *opts.NewListOptsRef(&newEnv, nil)
	containerConfig, err := parse(flags, copts)
	if err != nil {
		reportError(sudockerCli.Err(), "run", err.Error(), true)
		return cli.StatusError{StatusCode: 125}
	}
	return runContainer(ctx, sudockerCli, ropts, copts, containerConfig)
}

func runContainer(ctx context.Context, sudockerCli *cmd.SudockerCli, runOpts *runOptions, copts *containerOptions, containerCfg *containerConfig) error {
	config := containerCfg.Config
	stdout, stderr := sudockerCli.Out(), sudockerCli.Err()

	if !runOpts.detach {
		if err := sudockerCli.In().CheckTty(config.AttachStdin, config.Tty); err != nil {
			return err
		}
	} else {
		if copts.attach.Len() != 0 {
			return errors.Errorf("Conflicting options: -a and -d")
		}

		config.AttachStdin = false
		config.AttachStdout = false
		config.AttachStderr = false
		config.StdinOnce = false
	}
	ctx, cancelFun := context.WithCancel(ctx)
	defer cancelFun()
	parentProcess, err := newContainer(ctx, sudockerCli, containerCfg, &runOpts.createOptions)
	if err != nil {
		reportError(stderr, "run", err.Error(), true)
		return runStartContainerErr(err)
	}

	containerId := parentProcess.containerId
	hostConfig := containerCfg.HostConfig
	parent := parentProcess.cmd
	statusChan := make(chan string)
	go func() {
		if !config.Tty {
			_, _ = parent.Process.Wait()
			// 修改容器信息
			containerInfo, err := container.GetInfoByContainerId(containerId)
			if err != nil {
				logrus.Errorf("Get container %s info error %v", containerId, err)
			}
			containerInfo.Status = container.Stopped
			containerInfo.Pid = ""
			newContentBytes, err := json.Marshal(containerInfo)
			if err != nil {
				logrus.Errorf("Json marshal %s error %v", containerId, err)
			}
			dirPath := fmt.Sprintf(utils.InfoLocFormat, containerId)
			configFilePath := path.Join(dirPath, utils.ConfigName)
			if err = os.WriteFile(configFilePath, newContentBytes, 0o622); err != nil {
				logrus.Errorf("Write file %s error:%v", configFilePath, err)
			}
			if hostConfig.AutoRemove {
				statusChan <- "stopped"
			}
		}
		// --rm 在容器退出后清理容器
	}()
	if config.Tty {
		if err := parent.Wait(); err != nil {
			return errors.Errorf("Parent process failed: %v", err)
		}
		// 修改容器信息
		containerInfo, err := container.GetInfoByContainerId(containerId)
		if err != nil {
			return errors.Errorf("Get container %s info error %v", containerId, err)
		}
		containerInfo.Status = container.Stopped
		containerInfo.Pid = ""
		newContentBytes, err := json.Marshal(containerInfo)
		if err != nil {
			return errors.Errorf("Json marshal %s error %v", containerId, err)
		}
		dirPath := fmt.Sprintf(utils.InfoLocFormat, containerId)
		configFilePath := path.Join(dirPath, utils.ConfigName)
		if err = os.WriteFile(configFilePath, newContentBytes, 0o622); err != nil {
			return errors.Errorf("Write file %s error:%v", configFilePath, err)
		}
		if hostConfig.AutoRemove {
			statusChan <- "stopped"
		}
	}
	// logrus.Infof("auto remove: %v", copts.autoRemove)
	if hostConfig.AutoRemove {
		go func() {
			for {
				select {
				case status := <-statusChan:
					if status == "stopped" {
						containerInfo, err := container.GetInfoByContainerId(containerId)
						if err != nil {
							logrus.Errorf("Failed to get container info: %v", err)
							return
						}
						if containerInfo.Status == container.Stopped {
							// 容器已经停止，执行删除操作
							if err := container.DeleteStorageDriver(containerId, hostConfig.Binds); err != nil {
								logrus.Errorf("Umount volumes failed: %v", err)
							}
							if err := container.DeleteContainerInfo(containerId); err != nil {
								logrus.Errorf("Delete container info failed: %v", err)
							}
							parentProcess.cgroupManager.Destroy()
							return
						}
					}
				}
			}

		}()
	}

	//
	var (
		waitDisplayID chan struct{}
	)
	if !config.AttachStdout && !config.AttachStderr {
		// Make this asynchronous to allow the client to write to stdin before having to read the ID
		waitDisplayID = make(chan struct{})
		go func() {
			defer close(waitDisplayID)
			_, _ = fmt.Fprintln(stdout, containerId)
		}()
	}

	if (config.AttachStdin || config.AttachStdout || config.AttachStderr) && config.Tty && sudockerCli.Out().IsTerminal() {
		if err := MonitorTtySize(ctx, sudockerCli, containerId, false); err != nil {
			_, _ = fmt.Fprintln(stderr, "Error monitoring TTY size:", err)
		}
	}
	if !config.AttachStdout && !config.AttachStderr {
		// Detached mode
		<-waitDisplayID
		return nil
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
