package cgroups

import (
	"errors"
	"os"
	"strconv"

	"github.com/DeJeune/sudocker/runtime/config"
)

func isHugeTlbSet(r *config.Resources) bool {
	return len(r.HugetlbLimit) > 0
}

func setHugeTlb(dirPath string, r *config.Resources) error {
	if !isHugeTlbSet(r) {
		return nil
	}

	const suffix = ".max"
	skipRsvd := false
	for _, hugetlb := range r.HugetlbLimit {
		prefix := "hugetlb." + hugetlb.Pagesize
		val := strconv.FormatUint(hugetlb.Limit, 10)
		if err := WriteFile(dirPath, prefix+suffix, val); err != nil {
			return err
		}
		if skipRsvd {
			continue
		}

		if err := WriteFile(dirPath, prefix+".rsvd"+suffix, val); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				skipRsvd = true
				continue
			}
			return err
		}
	}
	return nil
}
