package fs

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/DeJeune/sudocker/runtime/pkg/cgroups"
	"github.com/DeJeune/sudocker/runtime/pkg/cgroups/fscommon"
	"golang.org/x/sys/unix"
)

type CpuSubsystem struct{}

func (s *CpuSubsystem) Name() string {
	return "cpu"
}

func (s *CpuSubsystem) Apply(path string, r *config.Resources, pid int) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	// 在进程迁移之前，我们应该先设置实时组调度配置，因为如果进程已经处于SCHED_RR模式且未设置RT带宽，那么此时添加将会失败
	if err := s.SetRtSched(path, r); err != nil {
		return err
	}
	// 由于我们没有使用apply()方法，我们需要将进程ID放入procs文件中。
	return cgroups.WriteCgroupProc(path, pid)
}

func (s *CpuSubsystem) SetRtSched(path string, r *config.Resources) error {
	var period string
	if r.CpuRtPeriod != 0 {
		period = strconv.FormatUint(r.CpuRtPeriod, 10)
		if err := cgroups.WriteFile(path, "cpu.rt_period_us", period); err != nil {
			// The values of cpu.rt_period_us and cpu.rt_runtime_us
			// are inter-dependent and need to be set in a proper order.
			// If the kernel rejects the new period value with EINVAL
			// and the new runtime value is also being set, let's
			// ignore the error for now and retry later.
			if !errors.Is(err, unix.EINVAL) || r.CpuRtRuntime == 0 {
				return err
			}
		} else {
			period = ""
		}
	}
	if r.CpuRtRuntime != 0 {
		if err := cgroups.WriteFile(path, "cpu.rt_runtime_us", strconv.FormatInt(r.CpuRtRuntime, 10)); err != nil {
			return err
		}
		if period != "" {
			if err := cgroups.WriteFile(path, "cpu.rt_period_us", period); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *CpuSubsystem) Set(path string, r *config.Resources) error {
	if r.CpuShares != 0 {
		shares := r.CpuShares
		if err := cgroups.WriteFile(path, "cpu.shares", strconv.FormatUint(shares, 10)); err != nil {
			return err
		}
		// read it back
		sharesRead, err := fscommon.GetCgroupParamUint(path, "cpu.shares")
		if err != nil {
			return err
		}
		// ... and check
		if shares > sharesRead {
			return fmt.Errorf("the maximum allowed cpu-shares is %d", sharesRead)
		} else if shares < sharesRead {
			return fmt.Errorf("the minimum allowed cpu-shares is %d", sharesRead)
		}
	}
	var period string
	if r.CpuPeriod != 0 {
		period = strconv.FormatUint(r.CpuPeriod, 10)
		if err := cgroups.WriteFile(path, "cpu.cfs_period_us", period); err != nil {
			// 有时，当要设置的周期小于当前周期时，内核会因旧配额/新周期超过父cgroup
			// 配额限制而拒绝（EINVAL）。如果发生这种情况且即将设置配额，请暂时忽略该
			// 错误并在设置配额后重试。
			if !errors.Is(err, unix.EINVAL) || r.CpuQuota == 0 {
				return err
			}
		} else {
			period = ""
		}
	}
	if r.CpuQuota != 0 {
		if err := cgroups.WriteFile(path, "cpu.cfs_quota_us", strconv.FormatInt(r.CpuQuota, 10)); err != nil {
			return err
		}
		if period != "" {
			if err := cgroups.WriteFile(path, "cpu.cfs_period_us", period); err != nil {
				return err
			}
		}
		// if burst != "" {
		// 	if err := cgroups.WriteFile(path, "cpu.cfs_burst_us", burst); err != nil {
		// 		return err
		// 	}
		// }
	}
	if r.CpuIdle != nil {
		idle := strconv.FormatInt(*r.CpuIdle, 10)
		if err := cgroups.WriteFile(path, "cpu.idle", idle); err != nil {
			return err
		}
	}

	return s.SetRtSched(path, r)
}
