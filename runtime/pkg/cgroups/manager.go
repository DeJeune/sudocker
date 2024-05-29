package cgroups

import (
	"fmt"
	"strings"

	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/DeJeune/sudocker/runtime/utils"
)

type BaseManager struct {
	config *config.Cgroup
	// like "/sys/fs/cgroup/user.slice/user-1001.slice"
	dirPath     string
	controllers map[string]struct{}
}

type Manager interface {
	// Apply creates a cgroup, if not yet created, and adds a process
	// with the specified pid into that cgroup.  A special value of -1
	// can be used to merely create a cgroup.
	Apply(pid int) error

	// GetPids returns the PIDs of all processes inside the cgroup.
	GetPids() ([]int, error)

	// GetAllPids returns the PIDs of all processes inside the cgroup
	// any all its sub-cgroups.
	GetAllPids() ([]int, error)

	// GetStats returns cgroups statistics.
	GetStats() (*Stats, error)

	// Freeze sets the freezer cgroup to the specified state.
	// Freeze(state configs.FreezerState) error

	// Destroy removes cgroup.
	Destroy() error

	// Path returns a cgroup path to the specified controller/subsystem.
	// For cgroupv2, the argument is unused and can be empty.
	Path(string) string

	// Set sets cgroup resources parameters/limits. If the argument is nil,
	// the resources specified during Manager creation (or the previous call
	// to Set) are used.
	Set(r *config.Resources) error

	// GetPaths returns cgroup path(s) to save in a state file in order to
	// restore later.
	//
	// For cgroup v1, a key is cgroup subsystem name, and the value is the
	// path to the cgroup for this subsystem.
	//
	// For cgroup v2 unified hierarchy, a key is "", and the value is the
	// unified path.
	GetPaths() map[string]string

	// GetCgroups returns the cgroup data as configured.
	GetCgroups() (*config.Cgroup, error)

	// GetFreezerState retrieves the current FreezerState of the cgroup.
	// GetFreezerState() (FreezerState, error)

	// Exists returns whether the cgroup path exists or not.
	Exists() bool

	// OOMKillCount reports OOM kill count for the cgroup.
	OOMKillCount() (uint64, error)

	// GetEffectiveCPUs returns the effective CPUs of the cgroup, an empty
	// value means that the cgroups cpuset subsystem/controller is not enabled.
	GetEffectiveCPUs() string
}

func NewManager(config *config.Cgroup, dirPath string) (*BaseManager, error) {
	if dirPath == "" {
		var err error
		dirPath, err = defaultDirPath(config)
		if err != nil {
			return nil, err
		}
	} else {
		dirPath = utils.CleanPath(dirPath)
	}

	m := &BaseManager{
		config:  config,
		dirPath: dirPath,
	}
	return m, nil
}

func (m *BaseManager) getControllers() error {
	if m.controllers != nil {
		return nil
	}

	data, err := ReadFile(m.dirPath, "cgroup.controllers")
	if err != nil {
		if m.config.Rootless && m.config.Path == "" {
			return nil
		}
		return err
	}
	fields := strings.Fields(data)
	m.controllers = make(map[string]struct{}, len(fields))
	for _, c := range fields {
		m.controllers[c] = struct{}{}
	}

	return nil
}

func CheckMemoryUsage(dirPath string, r *config.Resources) error {
	if !r.MemoryCheckBeforeUpdate {
		return nil
	}

	if r.Memory <= 0 && r.MemorySwap <= 0 {
		return nil
	}

	usage, err := GetCgroupParamUint(dirPath, "memory.current")
	if err != nil {
		// This check is on best-effort basis, so if we can't read the
		// current usage (cgroup not yet created, or any other error),
		// we should not fail.
		return nil
	}

	if r.MemorySwap > 0 {
		if uint64(r.MemorySwap) <= usage {
			return fmt.Errorf("rejecting memory+swap limit %d <= usage %d", r.MemorySwap, usage)
		}
	}

	if r.Memory > 0 {
		if uint64(r.Memory) <= usage {
			return fmt.Errorf("rejecting memory limit %d <= usage %d", r.Memory, usage)
		}
	}

	return nil
}
