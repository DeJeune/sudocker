package fs2

import (
	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/DeJeune/sudocker/runtime/pkg/cgroups"
)

func isPidsSet(r *config.Resources) bool {
	return r.PidsLimit != 0
}

func setPids(dirPath string, r *config.Resources) error {
	if !isPidsSet(r) {
		return nil
	}

	if val := numToStr(r.PidsLimit); val != "" {
		if err := cgroups.WriteFile(dirPath, "pids.max", val); err != nil {
			return err
		}
	}
	return nil
}
