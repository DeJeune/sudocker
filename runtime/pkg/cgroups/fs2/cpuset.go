package fs2

import (
	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/DeJeune/sudocker/runtime/pkg/cgroups"
)

func isCpusetSet(r *config.Resources) bool {
	return r.CpusetCpus != "" || r.CpusetMems != ""
}

func SetCpuset(dirPath string, r *config.Resources) error {
	if !isCpusetSet(r) {
		return nil
	}

	if r.CpusetCpus != "" {
		if err := cgroups.WriteFile(dirPath, "cpuset.cpus", r.CpusetCpus); err != nil {
			return err
		}
	}

	if r.CpusetMems != "" {
		if err := cgroups.WriteFile(dirPath, "cpuset.mems", r.CpusetMems); err != nil {
			return err
		}
	}
	return nil
}
