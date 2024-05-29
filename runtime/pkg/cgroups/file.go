package cgroups

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/DeJeune/sudocker/runtime/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	cgroupfsDir    = "/sys/fs/cgroup"
	cgroupfsPrefix = cgroupfsDir + "/"
)

var (
	// TestMode is set to true by unit tests that need "fake" cgroupfs.
	TestMode bool

	cgroupRootHandle *os.File
	prepOnce         sync.Once
	prepErr          error
	resolveFlags     uint64
)

// 根据dir和flag打开cgroup文件
// 如果不是cgroup文件，则返回错误
// dir+file构成文件的绝对路径
func OpenFile(dir, file string, flags int) (*os.File, error) {
	if dir == "" {
		return nil, fmt.Errorf("no directory specified for %s", file)
	}
	return openFile(dir, file, flags)
}

// 从dir目录下的Cgroup文件中读取数据，仅cgroups文件使用的函数
func ReadFile(dir, file string) (string, error) {
	fd, err := OpenFile(dir, file, unix.O_RDONLY)
	if err != nil {
		return "", err
	}
	defer fd.Close()
	var buf bytes.Buffer

	_, err = buf.ReadFrom(fd)
	return buf.String(), err
}

func WriteFile(dir, file, data string) error {
	fd, err := OpenFile(dir, file, unix.O_WRONLY)
	if err != nil {
		return err
	}
	defer fd.Close()
	if _, err := fd.WriteString(data); err != nil {
		return fmt.Errorf("failed to write %q: %w", data, err)
	}
	return nil
}

func prepareOpenat2() error {
	prepOnce.Do(func() {
		fd, err := unix.Openat2(-1, cgroupfsDir, &unix.OpenHow{
			Flags: unix.O_DIRECTORY | unix.O_PATH | unix.O_CLOEXEC,
		})
		if err != nil {
			prepErr = &os.PathError{Op: "openat2", Path: cgroupfsDir, Err: err}
			if err != unix.ENOSYS {
				logrus.Warnf("falling back to securejoin: %s", prepErr)
			} else {
				logrus.Debug("openat2 not available, falling back to securejoin")
			}
		}
		file := os.NewFile(uintptr(fd), cgroupfsDir)

		var st unix.Statfs_t
		if err := unix.Fstatfs(int(file.Fd()), &st); err != nil {
			prepErr = &os.PathError{Op: "statfs", Path: cgroupfsDir, Err: err}
			logrus.Warnf("falling back to securejoin: %s", prepErr)
			return
		}
		cgroupRootHandle = file
		resolveFlags = unix.RESOLVE_BENEATH | unix.RESOLVE_NO_MAGICLINKS
		if st.Type == unix.CGROUP2_SUPER_MAGIC {
			// cgroupv2 只有单挂载点
			resolveFlags |= unix.RESOLVE_NO_XDEV | unix.RESOLVE_NO_SYMLINKS
		}
	})
	return prepErr
}

func openFile(dir, file string, flags int) (*os.File, error) {
	// 默认权限
	mode := os.FileMode(0)
	// 如果为测试模式，且flags包含只写模式
	if TestMode && flags&os.O_WRONLY != 0 {
		// 存在则截断，不存在则创建
		flags |= os.O_TRUNC | os.O_CREATE
		mode = 0o600
	}
	path := path.Join(dir, utils.CleanPath(file))
	// 检查是否能使用openat2系统调用
	if prepareOpenat2() != nil {
		return openFallback(path, flags, mode)
	}
	relPath := strings.TrimPrefix(path, cgroupfsPrefix)
	if len(relPath) == len(path) {
		return openFallback(path, flags, mode)
	}

	fd, err := unix.Openat2(int(cgroupRootHandle.Fd()), relPath, &unix.OpenHow{
		Resolve: resolveFlags,
		Flags:   uint64(flags) | unix.O_CLOEXEC,
		Mode:    uint64(mode),
	})
	if err != nil {
		err = &os.PathError{Op: "openat2", Path: path, Err: err}
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}

var errNotCgroupfs = errors.New("not a cgroup file")

// Can be changed by unit tests.
var openFallback = openAndCheck

func openAndCheck(path string, flags int, mode os.FileMode) (*os.File, error) {
	fd, err := os.OpenFile(path, flags, mode)
	if err != nil {
		return nil, err
	}
	if TestMode {
		return fd, nil
	}
	// 检查这是一个cgroupfs文件
	var st unix.Statfs_t
	if err := unix.Fstatfs(int(fd.Fd()), &st); err != nil {
		_ = fd.Close()
		return nil, &os.PathError{Op: "statfs", Path: path, Err: err}
	}
	if st.Type != unix.CGROUP_SUPER_MAGIC && st.Type != unix.CGROUP2_SUPER_MAGIC {
		_ = fd.Close()
		return nil, &os.PathError{Op: "open", Path: path, Err: errNotCgroupfs}
	}

	return fd, nil
}
