package fs

import (
	"errors"
	"strconv"

	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/DeJeune/sudocker/runtime/pkg/cgroups"
	"golang.org/x/sys/unix"
)

const (
	cgroupMemorySwapLimit = "memory.memsw.limit_in_bytes"
	cgroupMemoryLimit     = "memory.limit_in_bytes"
	cgroupMemoryUsage     = "memory.usage_in_bytes"
	cgroupMemoryMaxUsage  = "memory.max_usage_in_bytes"
)

type MemorySubsystem struct{}

func (s *MemorySubsystem) Name() string {
	return "memory"
}

func (s *MemorySubsystem) Apply(path string, _ *config.Resources, pid int) error {
	return apply(path, pid)
}

func setMemory(path string, val int64) error {
	if val == 0 {
		return nil
	}
	err := cgroups.WriteFile(path, cgroupMemoryLimit, strconv.FormatInt(val, 10))
	if !errors.Is(err, unix.EBUSY) {
		return err
	}
	return nil
}

func setSwap(path string, val int64) error {
	if val == 0 {
		return nil
	}

	return cgroups.WriteFile(path, cgroupMemorySwapLimit, strconv.FormatInt(val, 10))
}

// func setMemoryAndSwap(path string, r *config.Resources) error {
// 	if r.Memory == -1 && r.MemorySwap == 0 {
// 		// Only set swap if it's enabled in kernel
// 		if cgroups.PathExists(filepath.Join(path, cgroupMemorySwapLimit)) {
// 			r.MemorySwap = -1
// 		}
// 	}

// 	if r.Memory != 0 && r.MemorySwap != 0 {

// 	}
// }

func (s *MemorySubsystem) Set(path string, r *config.Resources) error {
	if err := setMemory(path, r.Memory); err != nil {
		return err
	}
	return nil
}
