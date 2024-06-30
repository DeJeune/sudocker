package container

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	_ "github.com/DeJeune/sudocker/runtime/pkg/nsenter"
	"github.com/DeJeune/sudocker/runtime/utils"

	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/DeJeune/sudocker/runtime/pkg/container"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	EnvExecPid = "mydocker_pid"
	EnvExecCmd = "mydocker_cmd"
)

type ExecOptions struct {
	DetachKeys  string
	Interactive bool
	TTY         bool
	Detach      bool
	User        string
	Privileged  bool
	Workdir     string
	Command     []string
}

func NewExecCommand(sudockerCli *cmd.SudockerCli) *cobra.Command {
	var options ExecOptions

	cmd := &cobra.Command{
		Use:   "exec [OPTIONS] CONTAINER COMMAND [ARG...]",
		Short: "Run a command in a running container",
		Args:  cli.RequiresMinArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("exec the contaiener")
			// 如果环境变量存在，说明C代码已经运行过了，即setns系统调用已经执行了，这里就直接返回，避免重复执行
			if os.Getenv(EnvExecPid) != "" {
				logrus.Infof("pid callback pid %v", os.Getgid())
				return nil
			}
			containerIDorName := args[0]
			options.Command = args[1:]
			return RunExec(cmd.Context(), sudockerCli, containerIDorName, options)
		},
		Annotations: map[string]string{
			"category-top": "2",
			"aliases":      "docker container exec, docker exec",
		},
	}

	flags := cmd.Flags()
	flags.SetInterspersed(false)

	flags.StringVarP(&options.DetachKeys, "detach-keys", "", "", "Override the key sequence for detaching a container")
	flags.BoolVarP(&options.Interactive, "interactive", "i", false, "Keep STDIN open even if not attached")
	flags.BoolVarP(&options.TTY, "tty", "t", false, "Allocate a pseudo-TTY")
	flags.BoolVarP(&options.Detach, "detach", "d", false, "Detached mode: run command in the background")
	flags.StringVarP(&options.User, "user", "u", "", "Username or UID (format: <name|uid>[:<group|gid>])")
	flags.BoolVarP(&options.Privileged, "privileged", "", false, "Give extended privileges to the command")

	return cmd
}

func RunExec(ctx context.Context, sudockerCli *cmd.SudockerCli, containerIDorName string, options ExecOptions) error {
	pid, err := getPidByContainerId(containerIDorName)
	if err != nil {
		return errors.Errorf("Exec container getContainerPidByName %s error %v", containerIDorName, err)
	}
	execOptions, err := parseExec(options)
	if err != nil {
		return err
	}

	if !execOptions.Detach {
		if err := sudockerCli.In().CheckTty(execOptions.AttachStdin, execOptions.Tty); err != nil {
			return err
		}
	}

	fillConsoleSize(execOptions, sudockerCli)

	cmd := exec.Command("/proc/self/exe", "exec")

	if execOptions.AttachStdin {
		cmd.Stdin = sudockerCli.In()
	}
	if execOptions.AttachStdout {
		cmd.Stdout = sudockerCli.Out()
	}
	if execOptions.AttachStderr {
		if execOptions.Tty {
			cmd.Stderr = sudockerCli.Out()
		} else {
			cmd.Stderr = sudockerCli.Err()
		}
	}

	// 把命令拼接成字符串，便于传递
	cmdStr := strings.Join(execOptions.Cmd, " ")
	logrus.Infof("container pid: %s command: %s", pid, cmdStr)
	_ = os.Setenv(EnvExecPid, pid)
	_ = os.Setenv(EnvExecCmd, cmdStr)
	containerEnvs, err := getEnvsByPid(pid)
	if err != nil {
		return errors.Errorf("get env value failed: %v", err)
	}
	cmd.Env = append(os.Environ(), containerEnvs...)

	if err = cmd.Run(); err != nil {
		return errors.Errorf("Exec container %s error %v", containerIDorName, err)
	}

	if execOptions.Tty && sudockerCli.In().IsTerminal() {
		if err := MonitorTtySize(ctx, sudockerCli, strconv.Itoa(cmd.Process.Pid), true); err != nil {
			_, _ = fmt.Fprintln(sudockerCli.Err(), "Error monitoring TTY size:", err)
		}
	}
	return nil
}

func parseExec(execOpts ExecOptions) (*config.ExecOptions, error) {
	execOptions := &config.ExecOptions{
		User:       execOpts.User,
		Privileged: execOpts.Privileged,
		Tty:        execOpts.TTY,
		Cmd:        execOpts.Command,
		Detach:     execOpts.Detach,
		WorkingDir: execOpts.Workdir,
	}

	// If -d is not set, attach to everything by default
	if !execOpts.Detach {
		execOptions.AttachStdout = true
		execOptions.AttachStderr = true
		if execOpts.Interactive {
			execOptions.AttachStdin = true
		}
	}

	if execOpts.DetachKeys != "" {
		execOptions.DetachKeys = execOpts.DetachKeys
	}
	return execOptions, nil
}

func fillConsoleSize(execOptions *config.ExecOptions, sudockerCli cmd.Cli) {
	if execOptions.Tty {
		height, width := sudockerCli.Out().GetTtySize()
		execOptions.ConsoleSize = &[2]uint{height, width}
	}
}

func getPidByContainerId(containerId string) (string, error) {
	// 拼接出记录容器信息的文件路径
	dirPath := fmt.Sprintf(utils.InfoLocFormat, containerId)
	configFilePath := path.Join(dirPath, utils.ConfigName)
	// 读取内容并解析
	contentBytes, err := os.ReadFile(configFilePath)
	if err != nil {
		return "", err
	}
	var containerInfo container.Info
	if err = json.Unmarshal(contentBytes, &containerInfo); err != nil {
		return "", err
	}
	return containerInfo.Pid, nil
}

// getEnvsByPid 读取指定PID进程的环境变量
func getEnvsByPid(pid string) ([]string, error) {
	path := fmt.Sprintf("/proc/%s/environ", pid)
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Errorf("Read file %s error %v", path, err)
	}
	// env split by \u0000
	envs := strings.Split(string(contentBytes), "\u0000")
	return envs, nil
}
