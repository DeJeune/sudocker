package userns

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

type Mapping struct {
	UIDMappings []config.IDMap
	GIDMappings []config.IDMap
}

// toSys 函数将 Mapping 类型中的 UID 和 GID 映射转换为 syscall.SysProcIDMap 类型的切片。
// 这个转换过程包括遍历 Mapping 中的 UID 和 GID 映射，并将每个映射添加到相应的切片中。
// 返回的 uids 和 gids 切片可以用于系统调用，如设置用户命名空间的映射。
func (m Mapping) toSys() (uids, gids []syscall.SysProcIDMap) {
	for _, uid := range m.UIDMappings {
		uids = append(uids, syscall.SysProcIDMap{
			ContainerID: int(uid.ContainerID),
			HostID:      int(uid.HostID),
			Size:        int(uid.Size),
		})
	}
	for _, gid := range m.GIDMappings {
		gids = append(gids, syscall.SysProcIDMap{
			ContainerID: int(gid.ContainerID),
			HostID:      int(gid.HostID),
			Size:        int(gid.Size),
		})
	}
	return
}

func (m Mapping) id() string {
	var uids, gids []string
	for _, idmap := range m.UIDMappings {
		uids = append(uids, fmt.Sprintf("%d:%d:%d", idmap.ContainerID, idmap.HostID, idmap.Size))
	}
	for _, idmap := range m.GIDMappings {
		gids = append(gids, fmt.Sprintf("%d:%d:%d", idmap.ContainerID, idmap.HostID, idmap.Size))
	}
	sort.Strings(uids)
	sort.Strings(gids)
	return "uid=" + strings.Join(uids, ",") + ";gid=" + strings.Join(gids, ",")
}

type Handles struct {
	m    sync.Mutex
	maps map[string]*os.File
}

func (hs *Handles) Release() {
	hs.m.Lock()
	defer hs.m.Unlock()

	// Close the files for good measure, though GC will do that for us anyway.
	for _, file := range hs.maps {
		_ = file.Close()
	}
	hs.maps = nil
}

func (hs *Handles) Get(req Mapping) (file *os.File, err error) {
	hs.m.Lock()
	defer hs.m.Unlock()

	if hs.maps == nil {
		hs.maps = make(map[string]*os.File)
	}
	file, ok := hs.maps[req.id()]
	if !ok {
		proc, err := spawnProc(req)
		if err != nil {
			return nil, fmt.Errorf("failed to spawn dummy process for map %s: %w", req.id(), err)
		}
		defer func() {
			_ = proc.Kill()
			_, _ = proc.Wait()
		}()
		file, err = os.Open(fmt.Sprintf("/proc/%d/ns/user", proc.Pid))
		if err != nil {
			return nil, err
		}
		hs.maps[req.id()] = file
	}
	return dupFile(file)
}

// 生成一个子进程，为其设定uid,gid映射。这个子进程被置于usernamespace中
func spawnProc(req Mapping) (*os.Process, error) {
	logrus.Debugf("spawning dummy process for id-mapping %s", req.id())
	uidMappings, gidMappings := req.toSys()
	return os.StartProcess("/proc/self/exe", []string{"sudocker", "--help"}, &os.ProcAttr{
		Sys: &syscall.SysProcAttr{
			Cloneflags:                 unix.CLONE_NEWUSER,
			UidMappings:                uidMappings,
			GidMappings:                gidMappings,
			GidMappingsEnableSetgroups: false,
			// 不关心进程执行什么，只关心是否处于正确的命名空间
			Ptrace: true,
		},
	})
}

func dupFile(f *os.File) (*os.File, error) {
	newFd, err := unix.FcntlInt(f.Fd(), unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		return nil, os.NewSyscallError("fcntl(F_DUPFD_CLOEXEC)", err)
	}
	return os.NewFile(uintptr(newFd), f.Name()), nil
}
