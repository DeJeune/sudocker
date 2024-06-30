package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/DeJeune/sudocker/cli"
	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/DeJeune/sudocker/runtime/pkg/cgroups/fs"
	"github.com/DeJeune/sudocker/runtime/pkg/container"
	"github.com/DeJeune/sudocker/runtime/pkg/network"
	"github.com/DeJeune/sudocker/runtime/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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

type ParentProcess struct {
	cmd           *exec.Cmd
	containerId   string
	cgroupManager *fs.Manager
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

func runCreate(ctx context.Context, sudockerCli *cmd.SudockerCli, flags *pflag.FlagSet, options *createOptions, copts *containerOptions) error {
	containerConfig, err := parse(flags, copts)
	if err != nil {
		reportError(sudockerCli.Err(), "create", err.Error(), true)
		return cli.StatusError{StatusCode: 125}
	}
	parentProcess, err := newContainer(ctx, sudockerCli, containerConfig, options)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(sudockerCli.Out(), parentProcess.containerId)
	return nil
}

func newContainer(ctx context.Context, sudockerCli *cmd.SudockerCli, containerConfig *containerConfig, options *createOptions) (parentProcess *ParentProcess, err error) {
	cg := containerConfig.Config
	hostConfig := containerConfig.HostConfig
	networkConfig := containerConfig.NetworkingConfig
	// 模拟镜像拉取的过程
	if err := pullImage(ctx, sudockerCli, cg.Image, options); err != nil {
		return nil, err
	}
	containerId := container.GenerateContainerID()
	parentProcess = &ParentProcess{
		containerId: containerId,
	}
	parent, writePipe := container.NewParentProcess(ctx, sudockerCli, cg, hostConfig, containerId)
	parentProcess.cmd = parent
	if err := parent.Start(); err != nil {
		return nil, errors.Errorf("Failed to start parent process: %v", err)
	}

	info := &container.Info{
		Pid:         strconv.Itoa(parent.Process.Pid),
		Command:     strings.Join(cg.Cmd, ""),
		Created:     time.Now().Format("2006-01-02 15:04:05"),
		Id:          containerId,
		Name:        map[bool]string{true: containerId, false: options.name}[options.name == ""],
		Status:      container.Running,
		Volumes:     containerConfig.HostConfig.Binds,
		PortMapping: hostConfig.PortBindings,
	}
	// 如果没有指定网络，则分配默认网络sudocker0
	net := networkConfig.Endpoints
	logrus.Infof("portmap : %v", hostConfig.PortBindings)
	if net == "" {
		net = "sudocker0"
		res, err := network.ContainsNetwork(net)
		if err != nil {
			return parentProcess, err
		}
		if !res {
			if err := network.CreateNetwork("bridge", "172.17.0.0/16", "sudocker0"); err != nil {
				return parentProcess, err
			}
		}

	}
	ip, err := network.Connect(net, info)
	if err != nil {
		return parentProcess, errors.Errorf("Error Connect Network %v", err)
	}
	containerIP := ip.String()
	info.IP = containerIP

	if err := container.RecordContainerInfo(info); err != nil {
		return nil, err
	}

	cgroupConfig := &config.Cgroup{
		Name:      "sudocker-cgroup",
		Rootless:  false,
		Resources: hostConfig.Resources,
	}

	cgroupManager, err := fs.NewManager(cgroupConfig, nil)
	parentProcess.cgroupManager = cgroupManager
	// defer cgroupManager.Destroy()
	if err != nil {
		return nil, err
	}
	err = cgroupManager.Set(cgroupConfig.Resources)
	if err != nil {
		return nil, err
	}
	err = cgroupManager.Apply(parent.Process.Pid)
	if err != nil {
		return nil, err
	}
	sendInitCommand(cg.Cmd, writePipe)
	return parentProcess, nil
}

// sendInitCommand 通过writePipe将指令发送给子进程
func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	_, _ = writePipe.WriteString(command)
	_ = writePipe.Close()
}

func pullImage(ctx context.Context, sudockerCli *cmd.SudockerCli, img string, options *createOptions) error {
	// 打开源文件
	logrus.Infof("image name: %s", img)
	_, err := os.Stat(utils.ImagePath + img + ".tar")
	if os.IsNotExist(err) {
		sourceFile, err := os.Open(utils.ImagePath + "busybox.tar")
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		// 创建目标文件
		destinationFile, err := os.Create(utils.ImagePath + img + ".tar")
		if err != nil {
			return err
		}
		defer destinationFile.Close()

		// 复制文件内容
		_, err = io.Copy(destinationFile, sourceFile)
		if err != nil {
			return err
		}

		// 确保写入数据的完整性
		err = destinationFile.Sync()
		if err != nil {
			return err
		}
	}

	return nil
}
