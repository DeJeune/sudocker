package cgroups

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DeJeune/sudocker/runtime/config"
)

func supportedControllers() (string, error) {
	return ReadFile(UnifiedMountpoint, "/cgroup.controllers")
}

// needAnyControllers返回 是否使用一些支持的controller
func needAnyControllers(r *config.Resources) (bool, error) {
	if r == nil {
		return false, nil
	}

	content, err := supportedControllers()
	if err != nil {
		return false, err
	}
	avail := make(map[string]struct{})
	for _, ctr := range strings.Fields(content) {
		avail[ctr] = struct{}{}
	}

	have := func(controller string) bool {
		_, ok := avail[controller]
		return ok
	}

	if isCpuSet(r) && have("cpu") {
		return true, nil
	}

	if isCpusetSet(r) && have("cpuset") {
		return true, nil
	}

	if isMemorySet(r) && have("memory") {
		return true, nil
	}
	if isIoSet(r) && have("io") {
		return true, nil
	}

	if isPidsSet(r) && have("pids") {
		return true, nil
	}

	if isHugeTlbSet(r) && have("hugetlb") {
		return true, nil
	}

	return false, nil
}

func containsDomainController(r *config.Resources) bool {
	return isMemorySet(r) || isIoSet(r) || isCpuSet(r) || isHugeTlbSet(r)
}

func CreateCgroupPath(path string, c *config.Cgroup) (Err error) {
	if !strings.HasPrefix(path, UnifiedMountpoint) {
		return fmt.Errorf("invalid cgroup path %s", path)
	}

	content, err := supportedControllers()
	if err != nil {
		return err
	}

	const (
		cgTypeFile  = "cgroup.type"
		cgStCtlFile = "cgroup.subtree_control"
	)
	ctrs := strings.Fields(content)
	res := "+" + strings.Join(ctrs, " +")
	elements := strings.Split(path, "/")
	elements = elements[3:]
	current := "/sys/fs"
	for i, e := range elements {
		current = filepath.Join(current, e)
		if i > 0 {
			if err := os.Mkdir(current, 0o755); err != nil {
				if !os.IsExist(err) {
					return err
				}
			} else {
				current := current
				defer func() {
					if Err != nil {
						os.Remove(current)
					}
				}()
			}
			cgType, _ := ReadFile(current, cgTypeFile)
			cgType = strings.TrimSpace(cgType)
			switch cgType {
			case "domain invalid":
				if containsDomainController(c.Resources) {
					return fmt.Errorf("cannot enter cgroupv2 %q with domain controllers -- it is in an invalid state", current)
				} else {
					_ = WriteFile(current, cgTypeFile, "threaded")
				}
			case "domain threaded":
				fallthrough
			case "threaded":
				if containsDomainController(c.Resources) {
					return fmt.Errorf("cannot enter cgroupv2 %q with domain controllers -- it is in %s mode", current, cgType)
				}
			}

		}

		if i < len(elements)-1 {
			if err := WriteFile(current, cgStCtlFile, res); err != nil {
				// try write one by one
				allCtrs := strings.Split(res, " ")
				for _, ctr := range allCtrs {
					_ = WriteFile(current, cgStCtlFile, ctr)
				}
			}
			// Some controllers might not be enabled when rootless or containerized,
			// but we don't catch the error here. (Caught in setXXX() functions.)
		}
	}
	return nil
}
