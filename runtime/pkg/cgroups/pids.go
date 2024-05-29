package cgroups

import "github.com/DeJeune/sudocker/runtime/config"

func isPidsSet(r *config.Resources) bool {
	return r.PidsLimit != 0
}

func setPids(dirPath string, r *config.Resources) error {
	if !isPidsSet(r) {
		return nil
	}

	if val := numToStr(r.PidsLimit); val != "" {
		if err := WriteFile(dirPath, "pids.max", val); err != nil {
			return err
		}
	}
	return nil
}
