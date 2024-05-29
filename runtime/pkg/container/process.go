package container

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/DeJeune/sudocker/runtime/pkg/cgroups"
	"github.com/DeJeune/sudocker/runtime/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

var errInvalidProcess = errors.New("invalid process")

type pid struct {
	Pid           int `json:"stage2_pid"`
	PidFirstChild int `json:"stage1_pid"`
}

type processOperations interface {
	wait() (*os.ProcessState, error)
	signal(sig os.Signal) error
	pid() int
}

// 容器运行时创建临时的第一个父进程，用于为容器创建隔离的命名空间
type parentProcess interface {
	pid() int
	start() error
	// 中止命令
	terminate() error
	// wait waits on the process returning the process state.
	wait() (*os.ProcessState, error)
	// 与init进程通信
	signal(os.Signal) error
	// startTime returns the process start time.
	startTime() (uint64, error)
}

// 容器运行时实现进程间通信
type processComm struct {
	initSockParent *os.File
	initSockChild  *os.File
	syncSockParent *syncSocket
	syncSockChild  *syncSocket
}

func newProcessComm() (*processComm, error) {
	var (
		comm processComm
		err  error
	)
	comm.initSockParent, comm.initSockChild, err = utils.NewSockPair("init")
	if err != nil {
		return nil, fmt.Errorf("unable to create init pipe: %w", err)
	}
	return &comm, nil
}

func (c *processComm) closeChild() {
	_ = c.initSockChild.Close()
	_ = c.syncSockChild.Close()
	// _ = c.logPipeChild.Close()
}

func (c *processComm) closeParent() {
	_ = c.initSockParent.Close()
	_ = c.syncSockParent.Close()
	// c.logPipeParent is kept alive for ForwardLogs
}

// 管理命名空间
type setnsProcess struct {
	cmd            *exec.Cmd
	comm           *processComm
	cgroupPaths    map[string]string
	manager        cgroups.Manager
	intelRdtPath   string
	config         *initConfig
	container      *Container
	fds            []string
	process        *Process
	bootstrapData  io.Reader
	initProcessPid int
}

func (p *setnsProcess) start() (retErr error) {
	defer p.comm.closeParent()
	var err error
	err = p.cmd.Start()
	p.comm.closeChild()
	if err != nil {
		return fmt.Errorf("error starting setns process: %w", err)
	}
	waitInit := initWaiter(p.comm.initSockParent)
	defer func() {
		if retErr != nil {
			werr := <-waitInit
			if werr != nil {
				logrus.WithError(werr).Warn()
			}
			err := ignoreTerminateErrors(p.terminate())
			if err != nil {
				logrus.WithError(err).Warn("unable to terminate setnsProcess")
			}
		}
	}()
	if p.bootstrapData != nil {
		// 将bootstrapData通过管道传入
		if _, err := io.Copy(p.comm.initSockParent, p.bootstrapData); err != nil {
			return fmt.Errorf("error copying bootstrap data to pipe: %w", err)
		}
	}
	err = <-waitInit
	if err != nil {
		return err
	}

	if err := p.execSetns(); err != nil {
		return fmt.Errorf("error executing setns process: %w", err)
	}
	for _, path := range p.cgroupPaths {
		if err := cgroups.WriteCgroupProc(path, p.pid()); err != nil {
			if cgroups.IsCgroup2UnifiedMode() && p.initProcessPid != 0 {
				initProcCgroupFile := fmt.Sprintf("/proc/%d/cgroup", p.initProcessPid)
				initCg := make(map[string]string, 1)
				initCg[""] = initProcCgroupFile
				if initCgPath, ok := initCg[""]; ok {
					initCgDirpath := filepath.Join(cgroups.UnifiedMountpoint, initCgPath)
					logrus.Debugf("adding pid %d to cgroups %v failed (%v), attempting to join %q (obtained from %s)",
						p.pid(), p.cgroupPaths, err, initCg, initCgDirpath)
					// NOTE: initCgDirPath is not guaranteed to exist because we didn't pause the container.
					err = cgroups.WriteCgroupProc(initCgDirpath, p.pid())
				}
			}
		}
		if err != nil {
			return fmt.Errorf("error adding pid %d to cgroups: %w", p.pid(), err)
		}

	}

	if err := utils.WriteJSON(p.comm.initSockParent, p.config); err != nil {
		return fmt.Errorf("error writing config to pipe: %w", err)
	}

	var seenProcReady bool
	ierr := parseSync(p.comm.syncSockParent, func(sync *syncT) error {
		switch sync.Type {
		case procReady:
			seenProcReady = true
			if err := writeSync(p.comm.syncSockParent, procRun); err != nil {
				return err
			}
		case procHooks:
			// This shouldn't happen.
			panic("unexpected procHooks in setns")
		case procMountPlease:
			// This shouldn't happen.
			panic("unexpected procMountPlease in setns")
		case procSeccomp:
			// 暂时不处理Seccomp相关
			// if p.config.Config.Seccomp.ListenerPath == "" {
			// 	return errors.New("seccomp listenerPath is not set")
			// }
			if sync.Arg == nil {
				return fmt.Errorf("sync %q is missing an argument", sync.Type)
			}
			var srcFd int
			if err := json.Unmarshal(*sync.Arg, &srcFd); err != nil {
				return fmt.Errorf("sync %q passed invalid fd arg: %w", sync.Type, err)
			}
			seccompFd, err := pidGetFd(p.pid(), srcFd)
			if err != nil {
				return fmt.Errorf("sync %q get fd %d from child failed: %w", sync.Type, srcFd, err)
			}
			defer seccompFd.Close()
			if err := writeSync(p.comm.syncSockParent, procSeccompDone); err != nil {
				return err
			}
		default:
			return errors.New("invalid JSON payload from child")

		}
		return nil
	})

	if err := p.comm.syncSockParent.Shutdown(unix.SHUT_WR); err != nil && ierr == nil {
		return err
	}
	if !seenProcReady && ierr == nil {
		ierr = errors.New("procReady not received")
	}
	// Must be done after Shutdown so the child will exit and we can wait for it.
	if ierr != nil {
		_, _ = p.wait()
		return ierr
	}
	return nil
}

func (p *setnsProcess) execSetns() error {
	status, err := p.cmd.Process.Wait()
	if err != nil {
		_ = p.cmd.Wait()
		return fmt.Errorf("error waiting on setns process to finish: %w", err)
	}
	if !status.Success() {
		_ = p.cmd.Wait()
		return &exec.ExitError{ProcessState: status}
	}
	var pid *pid
	if err := json.NewDecoder(p.comm.initSockParent).Decode(&pid); err != nil {
		_ = p.cmd.Wait()
		return fmt.Errorf("error reading pid from init pipe: %w", err)
	}
	firstChildProcess, _ := os.FindProcess(pid.PidFirstChild)

	// Ignore the error in case the child has already been reaped for any reason
	_, _ = firstChildProcess.Wait()

	process, err := os.FindProcess(pid.Pid)
	if err != nil {
		return err
	}
	p.cmd.Process = process
	p.process.ops = p
	return nil
}

func (p *setnsProcess) signal(sig os.Signal) error {
	s, ok := sig.(unix.Signal)
	if !ok {
		return errors.New("os: unsupported signal type")
	}
	return unix.Kill(p.pid(), s)
}

func (p *setnsProcess) terminate() error {
	if p.cmd.Process == nil {
		return nil
	}
	err := p.cmd.Process.Kill()
	if _, werr := p.wait(); err == nil {
		err = werr
	}
	return err
}

func (p *setnsProcess) wait() (*os.ProcessState, error) {
	err := p.cmd.Wait()

	// Return actual ProcessState even on Wait error
	return p.cmd.ProcessState, err
}

func (p *setnsProcess) pid() int {
	return p.cmd.Process.Pid
}

type initProcess struct {
	cmd           *exec.Cmd
	comm          *processComm
	config        *initConfig
	manager       cgroups.Manager
	container     *Container
	fds           []string
	process       *Process
	bootstrapData io.Reader
}

func (p *initProcess) pid() int {
	return p.cmd.Process.Pid
}

func (p *initProcess) externalDescriptors() []string {
	return p.fds
}

func (p *initProcess) getChildPid() (int, error) {
	var pid pid
	if err := json.NewDecoder(p.comm.initSockParent).Decode(&pid); err != nil {
		_ = p.cmd.Wait()
		return -1, err
	}
	firstChildProcess, _ := os.FindProcess(pid.PidFirstChild)
	_, _ = firstChildProcess.Wait()
	return pid.Pid, nil
}

func (p *initProcess) waitForChildExit(childPid int) error {
	status, err := p.cmd.Process.Wait()
	if err != nil {
		_ = p.cmd.Wait()
	}

	if !status.Success() {
		_ = p.cmd.Wait()
		return &exec.ExitError{ProcessState: status}
	}

	process, err := os.FindProcess(childPid)
	if err != nil {
		return err
	}
	p.cmd.Process = process
	p.process.ops = p
	return nil
}

func (p *initProcess) start() (retErr error) {
	defer p.comm.closeParent()
	err := p.cmd.Start()
	p.process.ops = p
	p.comm.closeChild()
}

func (p *initProcess) signal(sig os.Signal) error {
	s, ok := sig.(unix.Signal)
	if !ok {
		return errors.New("os: unsupported signal type")
	}
	return unix.Kill(p.pid(), s)
}

func (p *initProcess) wait() (*os.ProcessState, error) {
	err := p.cmd.Wait()
	return p.cmd.ProcessState, err
}

func (p *initProcess) terminate() error {
	if p.cmd.Process == nil {
		return nil
	}
	err := p.cmd.Process.Kill()
	if _, werr := p.wait(); err == nil {
		err = werr
	}
	return err
}

// sudocker run option1 container -- 容器执行的第一个进程
type Process struct {
	//参数
	Args []string
	//环境变量
	Env []string
	// 设置进程的uid,gid
	User     string
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
	Init     bool
	Loglevel string
	ops      processOperations
}

func (p Process) Wait() (*os.ProcessState, error) {
	if p.ops == nil {
		return nil, errInvalidProcess
	}
	return p.ops.wait()
}

func (p Process) Pid() (int, error) {
	if p.ops == nil {
		return math.MinInt32, errInvalidProcess
	}
	return p.ops.pid(), nil
}

func (p Process) Signal(sig os.Signal) error {
	if p.ops == nil {
		return errInvalidProcess
	}
	return p.ops.signal(sig)
}

type IO struct {
	Stdin  io.WriteCloser
	Stdout io.ReadCloser
	Stderr io.ReadCloser
}

func initWaiter(r io.Reader) chan error {
	ch := make(chan error, 1)
	go func() {
		defer close(ch)

		inited := make([]byte, 1)
		n, err := r.Read(inited)
		if err == nil {
			if n < 1 {
				err = errors.New("short read")
			} else if inited[0] != 0 {
				err = fmt.Errorf("unexpected %d != 0", inited[0])
			} else {
				ch <- nil
				return
			}
		}
		ch <- fmt.Errorf("waiting for init preliminary setup: %w", err)
	}()

	return ch
}

func ignoreTerminateErrors(err error) error {
	if err == nil {
		return nil
	}
	// terminate() might return an error from either Kill or Wait.
	// The (*Cmd).Wait documentation says: "If the command fails to run
	// or doesn't complete successfully, the error is of type *ExitError".
	// Filter out such errors (like "exit status 1" or "signal: killed").
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil
	}
	if errors.Is(err, os.ErrProcessDone) {
		return nil
	}
	s := err.Error()
	if strings.Contains(s, "Wait was already called") {
		return nil
	}
	return err
}

func pidGetFd(pid, srcFd int) (*os.File, error) {
	pidFd, err := unix.PidfdOpen(pid, 0)
	if err != nil {
		return nil, os.NewSyscallError("pidfd_open", err)
	}
	defer unix.Close(pidFd)
	fd, err := unix.PidfdGetfd(pidFd, srcFd, 0)
	if err != nil {
		return nil, os.NewSyscallError("pidfd_getfd", err)
	}
	return os.NewFile(uintptr(fd), "[pidfd_getfd]"), nil
}
