package fs2

import (
	"fmt"
	"strings"

	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/DeJeune/sudocker/runtime/pkg/cgroups"
	"github.com/DeJeune/sudocker/runtime/pkg/cgroups/fscommon"
	"github.com/DeJeune/sudocker/runtime/utils"
)

type BaseManager struct {
	config *config.Cgroup
	// like "/sys/fs/cgroup/user.slice/user-1001.slice"
	dirPath     string
	controllers map[string]struct{}
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

	data, err := cgroups.ReadFile(m.dirPath, "cgroup.controllers")
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

	usage, err := fscommon.GetCgroupParamUint(dirPath, "memory.current")
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
