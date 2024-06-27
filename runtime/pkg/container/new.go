package container

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/sirupsen/logrus"
)

func NewParentProcess(ctx context.Context, cli cmd.Cli, config *config.Config, hostConfig *config.HostConfig, containerId string) (*exec.Cmd, *os.File) {
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		logrus.Errorf("New pipe error %v", err)
		return nil, nil
	}
	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	if config.Tty {
		cmd.Stdin = cli.In()
		cmd.Stdout = cli.Out()
		cmd.Stderr = cli.Err()
	} else {
		// 对于后台运行容器，将 stdout、stderr 重定向到日志文件中，便于后续查看
		dirPath := fmt.Sprintf(InfoLocFormat, containerId)
		if err := os.MkdirAll(dirPath, 0o777); err != nil {
			logrus.Errorf("NewParentProcess mkdir %s error %v", dirPath, err)
			return nil, nil
		}
		stdLogFilePath := dirPath + LogFile
		stdLogFile, err := os.Create(stdLogFilePath)
		if err != nil {
			logrus.Errorf("NewParentProcess create file %s error %v", stdLogFilePath, err)
			return nil, nil
		}
		cmd.Stdout = stdLogFile
		cmd.Stderr = stdLogFile
	}
	cmd.ExtraFiles = []*os.File{readPipe}
	mntURL := "/root/merged/"
	rootURL := "/root/"
	if err := NewStorageDriver(rootURL, mntURL, hostConfig.Binds); err != nil {
		logrus.Errorf("mount storage driver failed: %v", err)
		return nil, nil
	}
	cmd.Dir = mntURL
	return cmd, writePipe
}
