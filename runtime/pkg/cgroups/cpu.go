package cgroups

import (
	"errors"
	"strconv"

	"github.com/DeJeune/sudocker/runtime/config"
	"golang.org/x/sys/unix"
)

const (
	cpu     = "cpu"
	cpuIdle = "cpu.idle"
	cpuMax  = "cpu.max"
)

func isCpuSet(r *config.Resources) bool {
	return r.CpuWeight != 0 || r.CpuQuota != 0 || r.CpuPeriod != 0 || r.CpuIdle != nil || r.CpuBurst != nil
}

func setCpu(dirPath string, r *config.Resources) error {
	if !isCpuSet(r) {
		return nil
	}
	if r.CpuIdle != nil {
		if err := WriteFile(dirPath, "cpu.idle", strconv.FormatInt(*r.CpuIdle, 10)); err != nil {
			return err
		}
	}

	if r.CpuWeight != 0 {
		if err := WriteFile(dirPath, "cpu.weight", strconv.FormatUint(r.CpuWeight, 10)); err != nil {
			return err
		}
	}

	var burst string
	if r.CpuBurst != nil {
		burst = strconv.FormatUint(*r.CpuBurst, 10)
		if err := WriteFile(dirPath, "cpu.max.burst", burst); err != nil {
			if !errors.Is(err, unix.EINVAL) || r.CpuQuota == 0 {
				return err
			}
		} else {
			burst = ""
		}
	}

	if r.CpuQuota != 0 || r.CpuPeriod != 0 {
		str := "max"
		if r.CpuQuota > 0 {
			str = strconv.FormatInt(r.CpuQuota, 10)
		}
		period := r.CpuPeriod
		if period == 0 {
			// 默认值
			// A read-write two value file which exists on non-root cgroups. The default is “max 100000”.
			period = 100000
		}
		str += " " + strconv.FormatUint(period, 10)
		if err := WriteFile(dirPath, "cpu.max", str); err != nil {
			return err
		}
		if burst != "" {
			if err := WriteFile(dirPath, "cpu.max.burst", burst); err != nil {
				return err
			}
		}
	}

	return nil
}

// func statCpu(dirPath string, stats *cgroups.Stats) error {
// 	const file_name = "cpu.stat"
// 	f, err := OpenFile(dirPath, file_name, os.O_RDONLY)
// 	if err != nil {
// 		return err
// 	}
// 	defer f.Close()

// 	sc := {

// 	} bufio.NewScanner(f)
// 	for sc.Scan()
// }
