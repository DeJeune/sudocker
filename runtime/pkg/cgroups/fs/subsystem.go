package fs

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/DeJeune/sudocker/runtime/pkg/cgroups"
	"golang.org/x/sys/unix"
)

type Subsystem interface {
	// Name 返回当前Subsystem的名称,比如cpu、memory
	Name() string
	// Set 设置某个cgroup在这个Subsystem中的资源限制
	Set(path string, res *config.Resources) error
	// Apply 将进程添加到某个cgroup中
	Apply(path string, r *config.Resources, pid int) error
	// Remove 移除某个cgroup
	// Remove(path string) error
	// 获取子系统统计数据
	// GetStats(path string, stats *cgroups.Stats) error
}

var errSubsystemDoesNotExist = errors.New("cgroup: subsystem does not exist")

type Manager struct {
	mu      sync.Mutex
	cgroups *config.Cgroup
	paths   map[string]string
}

var Subsystems = []Subsystem{
	&CpusetSubsystem{},
	&MemorySubsystem{},
	&CpuSubsystem{},
}

func NewManager(cg *config.Cgroup, paths map[string]string) (*Manager, error) {
	if cg.Resources == nil {
		return nil, errors.New("cgroup v1 manager needs configs.Resources to be set during manager creation")
	}
	if cg.Resources.Unified != nil {
		return nil, cgroups.ErrV1NoUnified
	}

	if paths == nil {
		var err error
		paths, err = initPaths(cg)
		if err != nil {
			return nil, err
		}
	}
	return &Manager{
		cgroups: cg,
		paths:   paths,
	}, nil
}

func (m *Manager) Apply(pid int) (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c := m.cgroups

	for _, sys := range Subsystems {
		name := sys.Name()
		p, ok := m.paths[name]
		if !ok {
			continue
		}

		if err := sys.Apply(p, c.Resources, pid); err != nil {
			// 在Rootless（包括在用户命名空间中euid=0）的情况下，如果没有明确设置
			// cgroup 路径，我们不会因权限问题而在这里退出。相反，我们会从m.paths映
			// 射中删除该路径，因为该路径要么不存在且无法创建，要么进程无法添加到其中。
			// 对于已设置子系统限制的情况，由Set稍后处理，它会友好地返回错误（参见Set
			// 中的if path == ""条件）。
			if isIgnorableError(c.Rootless, err) && c.Path == "" {
				delete(m.paths, name)
				continue
			}
			return err
		}

	}
	return nil
}

func (m *Manager) Destroy() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return cgroups.RemovePaths(m.paths)
}

func (m *Manager) Path(subsys string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.paths[subsys]
}

func (m *Manager) Set(r *config.Resources) error {
	if r == nil {
		return nil
	}

	if r.Unified != nil {
		return cgroups.ErrV1NoUnified
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for _, sys := range Subsystems {
		path := m.paths[sys.Name()]
		if err := sys.Set(path, r); err != nil {
			// When rootless is true, errors from the device subsystem
			// are ignored, as it is really not expected to work.
			if m.cgroups.Rootless && sys.Name() == "devices" && !errors.Is(err, cgroups.ErrDevicesUnsupported) {
				continue
			}
			// However, errors from other subsystems are not ignored.
			// see @test "runc create (rootless + limits + no cgrouppath + no permission) fails with informative error"
			if path == "" {
				// We never created a path for this cgroup, so we cannot set
				// limits for it (though we have already tried at this point).
				return fmt.Errorf("cannot set %s limit: container could not join or create cgroup", sys.Name())
			}
			return err
		}
	}

	return nil
}

func (m *Manager) GetCgroups() (*config.Cgroup, error) {
	return m.cgroups, nil
}

func isIgnorableError(rootless bool, err error) bool {
	// We do not ignore errors if we are root.
	if !rootless {
		return false
	}
	// Is it an ordinary EPERM?
	if errors.Is(err, os.ErrPermission) {
		return true
	}
	// Handle some specific syscall errors.
	var errno unix.Errno
	if errors.As(err, &errno) {
		return errno == unix.EROFS || errno == unix.EPERM || errno == unix.EACCES
	}
	return false
}
