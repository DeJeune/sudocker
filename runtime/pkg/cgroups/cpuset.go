package cgroups

import "github.com/DeJeune/sudocker/runtime/config"

func isCpusetSet(r *config.Resources) bool {
	return r.CpusetCpus != "" || r.CpusetMems != ""
}

func SetCpuset(dirPath string, r *config.Resources) error {
	if !isCpusetSet(r) {
		return nil
	}

	if r.CpusetCpus != "" {
		if err := WriteFile(dirPath, "cpuset.cpus", r.CpusetCpus); err != nil {
			return err
		}
	}

	if r.CpusetMems != "" {
		if err := WriteFile(dirPath, "cpuset.mems", r.CpusetMems); err != nil {
			return err
		}
	}
	return nil
}
