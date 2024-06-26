package container

import (
	"context"
	"os"
	"os/exec"
	"syscall"

	"github.com/DeJeune/sudocker/cmd"
	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/sirupsen/logrus"
)

func NewParentProcess(ctx context.Context, cli cmd.Cli, config *config.Config) (*exec.Cmd, *os.File) {
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
	}
	cmd.ExtraFiles = []*os.File{readPipe}
	return cmd, writePipe
}
