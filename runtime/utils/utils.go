package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "unsafe" // for go:linkname

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func RandStringBytes(n int) string {
	letterBytes := "1234567890"
	rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func EnsureProcHandle(fh *os.File) error {
	var buf unix.Statfs_t
	if err := unix.Fstatfs(int(fh.Fd()), &buf); err != nil {
		return fmt.Errorf("ensure %s is on procfs: %w", fh.Name(), err)
	}
	if buf.Type != unix.PROC_SUPER_MAGIC {
		return fmt.Errorf("%s is not on procfs", fh.Name())
	}
	return nil
}

type ProcThreadSelfCloser func()

var (
	haveProcThreadSelf     bool
	haveProcThreadSelfOnce sync.Once
)

// ProcThreadSelf returns a string that is equivalent to
// /proc/thread-self/<subpath>, with a graceful fallback on older kernels where
// /proc/thread-self doesn't exist. This method DOES NOT use SecureJoin,
// meaning that the passed string needs to be trusted. The caller _must_ call
// the returned procThreadSelfCloser function (which is runtime.UnlockOSThread)
// *only once* after it has finished using the returned path string.
func ProcThreadSelf(subpath string) (string, ProcThreadSelfCloser) {
	haveProcThreadSelfOnce.Do(func() {
		if _, err := os.Stat("/proc/thread-self/"); err == nil {
			haveProcThreadSelf = true
		} else {
			logrus.Debugf("cannot stat /proc/thread-self (%v), falling back to /proc/self/task/<tid>", err)
		}
	})

	// We need to lock our thread until the caller is done with the path string
	// because any non-atomic operation on the path (such as opening a file,
	// then reading it) could be interrupted by the Go runtime where the
	// underlying thread is swapped out and the original thread is killed,
	// resulting in pull-your-hair-out-hard-to-debug issues in the caller. In
	// addition, the pre-3.17 fallback makes everything non-atomic because the
	// same thing could happen between unix.Gettid() and the path operations.
	//
	// In theory, we don't need to lock in the atomic user case when using
	// /proc/thread-self/, but it's better to be safe than sorry (and there are
	// only one or two truly atomic users of /proc/thread-self/).
	runtime.LockOSThread()

	threadSelf := "/proc/thread-self/"
	if !haveProcThreadSelf {
		// Pre-3.17 kernels did not have /proc/thread-self, so do it manually.
		threadSelf = "/proc/self/task/" + strconv.Itoa(unix.Gettid()) + "/"
		if _, err := os.Stat(threadSelf); err != nil {
			// Unfortunately, this code is called from rootfs_linux.go where we
			// are running inside the pid namespace of the container but /proc
			// is the host's procfs. Unfortunately there is no real way to get
			// the correct tid to use here (the kernel age means we cannot do
			// things like set up a private fsopen("proc") -- even scanning
			// NSpid in all of the tasks in /proc/self/task/*/status requires
			// Linux 4.1).
			//
			// So, we just have to assume that /proc/self is acceptable in this
			// one specific case.
			if os.Getpid() == 1 {
				logrus.Debugf("/proc/thread-self (tid=%d) cannot be emulated inside the initial container setup -- using /proc/self instead: %v", unix.Gettid(), err)
			} else {
				// This should never happen, but the fallback should work in most cases...
				logrus.Warnf("/proc/thread-self could not be emulated for pid=%d (tid=%d) -- using more buggy /proc/self fallback instead: %v", os.Getpid(), unix.Gettid(), err)
			}
			threadSelf = "/proc/self/"
		}
	}
	return threadSelf + subpath, runtime.UnlockOSThread
}

// ProcThreadSelfFd is small wrapper around ProcThreadSelf to make it easier to
// create a /proc/thread-self handle for given file descriptor.
//
// It is basically equivalent to ProcThreadSelf(fmt.Sprintf("fd/%d", fd)), but
// without using fmt.Sprintf to avoid unneeded overhead.
func ProcThreadSelfFd(fd uintptr) (string, ProcThreadSelfCloser) {
	return ProcThreadSelf("fd/" + strconv.FormatUint(uint64(fd), 10))
}

// WriteJSON writes the provided struct v to w using standard json marshaling
// without a trailing newline. This is used instead of json.Encoder because
// there might be a problem in json decoder in some cases, see:
// https://github.com/docker/docker/issues/14203#issuecomment-174177790
func WriteJSON(w io.Writer, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func CleanPath(path string) string {
	if path == "" {
		return ""
	}

	path = filepath.Clean(path)

	if !filepath.IsAbs(path) {
		path = filepath.Clean(string(os.PathSeparator) + path)
		path, _ = filepath.Rel(string(os.PathSeparator), path)
	}
	return filepath.Clean(path)
}

func NewSockPair(name string) (parent, child *os.File, err error) {
	// 创建一个套接字对，AF_LOCAL本地套接字，unix.SOCK_STREAM|unix.SOCK_CLOEXEC 指定
	//套接字类型为流式套接字，并设置 CLOEXEC 标志（在执行新程序时关闭套接字）
	fds, err := unix.Socketpair(unix.AF_LOCAL, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, nil, err
	}
	return os.NewFile(uintptr(fds[1]), name+"-p"), os.NewFile(uintptr(fds[0]), name+"-c"), nil
}

// Annotations 从 libcontainer 状态返回包路径和用户定义的注释。我们需要删除包，因为这是 libcontainer 添加的标签。
func Annotations(labels []string) (bundle string, userAnnotations map[string]string) {
	userAnnotations = make(map[string]string)
	for _, l := range labels {
		parts := strings.SplitN(l, "=", 2)
		if len(parts) < 2 {
			continue
		}
		if parts[0] == "bundle" {
			bundle = parts[1]
		} else {
			userAnnotations[parts[0]] = parts[1]
		}
	}
	return
}

var (
	haveCloseRangeCloexecBool bool
	haveCloseRangeCloexecOnce sync.Once
)

func haveCloseRangeCloexec() bool {
	haveCloseRangeCloexecOnce.Do(func() {
		// 创建一个临时文件描述符 tmpFd。
		// F_DUPFD_CLOEXEC 标志用于复制文件描述符并设置 close-on-exec 标志。
		tmpFd, err := unix.FcntlInt(0, unix.F_DUPFD_CLOEXEC, 0)
		if err != nil {
			return
		}
		defer unix.Close(tmpFd)

		err = unix.CloseRange(uint(tmpFd), uint(tmpFd), unix.CLOSE_RANGE_CLOEXEC)
		// Any error means we cannot use close_range(CLOSE_RANGE_CLOEXEC).
		// -ENOSYS and -EINVAL ultimately mean we don't have support, but any
		// other potential error would imply that even the most basic close
		// operation wouldn't work.
		haveCloseRangeCloexecBool = err == nil
	})
	return haveCloseRangeCloexecBool
}

type fdFunc func(fd int)

// 遍历当前进程中打开的所有文件描述符，并对每个文件描述符执行给定的函数fn
func fdRangeFrom(minFd int, fn fdFunc) error {
	procSelfFd, closer := ProcThreadSelf("fd")
	defer closer()

	fdDir, err := os.Open(procSelfFd)
	if err != nil {
		return err
	}
	defer fdDir.Close()
	// 确保是一个proc文件系统
	if err := EnsureProcHandle(fdDir); err != nil {
		return err
	}
	// 读取目录中的所有文件名
	fdList, err := fdDir.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, fdStr := range fdList {
		fd, err := strconv.Atoi(fdStr)
		if err != nil {
			continue
		}
		if fd < minFd {
			continue
		}
		if uintptr(fd) == fdDir.Fd() {
			continue
		}
		fn(fd)
	}
	return nil
}

func CloseExecFrom(minFd int) error {
	if haveCloseRangeCloexec() {
		err := unix.CloseRange(uint(minFd), math.MaxUint, unix.CLOSE_RANGE_CLOEXEC)
		return os.NewSyscallError("close_range", err)
	}
	return fdRangeFrom(minFd, unix.CloseOnExec)
}

func stripRoot(root, path string) string {
	// Make the paths clean and absolute.
	root, path = CleanPath("/"+root), CleanPath("/"+path)
	switch {
	case path == root:
		path = "/"
	case root == "/":
		// do nothing
	case strings.HasPrefix(path, root+"/"):
		path = strings.TrimPrefix(path, root+"/")
	}
	return CleanPath("/" + path)
}

// WithProcfd 在根目录内解析的不安全路径对应的 procfd 路径（/proc/self/fd/...）上运行传入的闭包。
// 在传递 fd 之前，会验证此路径是否位于根目录内——因此通过传入的 fdpath 对其进行操作应该是安全的。
// 不要通过原始的路径字符串访问此路径，也不要尝试在传入的闭包之外使用该路径名（文件句柄将在闭包返回后释放）。
func WithProcfd(root, unsafePath string, fn func(procfd string) error) error {
	unsafePath = stripRoot(root, unsafePath)
	path, err := securejoin.SecureJoin(root, unsafePath)
	if err != nil {
		return fmt.Errorf("resolving path inside rootfs failed: %w", err)
	}
	procSelfFd, closer := ProcThreadSelf("fd/")
	defer closer()
	fh, err := os.OpenFile(path, unix.O_PATH|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open o_path procfd: %w", err)
	}
	defer fh.Close()

	procfd := filepath.Join(procSelfFd, strconv.Itoa(int(fh.Fd())))
	// Double-check the path is the one we expected.
	if realpath, err := os.Readlink(procfd); err != nil {
		return fmt.Errorf("procfd verification failed: %w", err)
	} else if realpath != path {
		return fmt.Errorf("possibly malicious path detected -- refusing to operate on %s", realpath)
	}

	return fn(procfd)
}

//go:linkname runtime_IsPollDescriptor internal/poll.IsPollDescriptor

// In order to make sure we do not close the internal epoll descriptors the Go
// runtime uses, we need to ensure that we skip descriptors that match
// "internal/poll".IsPollDescriptor. Yes, this is a Go runtime internal thing,
// unfortunately there's no other way to be sure we're only keeping the file
// descriptors the Go runtime needs. Hopefully nothing blows up doing this...
func runtime_IsPollDescriptor(fd uintptr) bool

// UnsafeCloseFrom closes all file descriptors greater or equal to minFd in the
// current process, except for those critical to Go's runtime (such as the
// netpoll management descriptors).
//
// NOTE: That this function is incredibly dangerous to use in most Go code, as
// closing file descriptors from underneath *os.File handles can lead to very
// bad behaviour (the closed file descriptor can be re-used and then any
// *os.File operations would apply to the wrong file). This function is only
// intended to be called from the last stage of runc init.
func UnsafeCloseFrom(minFd int) error {
	// We cannot use close_range(2) even if it is available, because we must
	// not close some file descriptors.
	return fdRangeFrom(minFd, func(fd int) {
		if runtime_IsPollDescriptor(uintptr(fd)) {
			// These are the Go runtimes internal netpoll file descriptors.
			// These file descriptors are operated on deep in the Go scheduler,
			// and closing those files from underneath Go can result in panics.
			// There is no issue with keeping them because they are not
			// executable and are not useful to an attacker anyway. Also we
			// don't have any choice.
			return
		}
		// There's nothing we can do about errors from close(2), and the
		// only likely error to be seen is EBADF which indicates the fd was
		// already closed (in which case, we got what we wanted).
		_ = unix.Close(fd)
	})
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func GetRoot(containerId string) string { return RootPath + containerId }

func GetImage(imageName string) string { return fmt.Sprintf("%s%s.tar", ImagePath, imageName) }

func GetLower(containerId string) string {
	return fmt.Sprintf(lowerDirFormat, containerId)
}

func GetUpper(containerId string) string {
	return fmt.Sprintf(upperDirFormat, containerId)
}

func GetWorker(containerId string) string {
	return fmt.Sprintf(workDirFormat, containerId)
}

func GetMerged(containerId string) string { return fmt.Sprintf(mergedDirFormat, containerId) }

func GetOverlayFSDirs(lower, upper, worker string) string {
	return fmt.Sprintf(overlayFSFormat, lower, upper, worker)
}

// GetLogfile build logfile name by containerId
func GetLogfile(containerId string) string {
	return fmt.Sprintf(LogFile, containerId)
}
