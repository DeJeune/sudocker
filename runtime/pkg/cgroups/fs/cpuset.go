package fs

import (
	"os"
	"path/filepath"

	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/DeJeune/sudocker/runtime/pkg/cgroups"
	"golang.org/x/sys/unix"
)

type CpusetSubsystem struct{}

func (s *CpusetSubsystem) Name() string {
	return "cpuset"
}

func (s *CpusetSubsystem) Apply(path string, r *config.Resources, pid int) error {
	return s.ApplyDir(path, r, pid)
}

func (s *CpusetSubsystem) Set(path string, r *config.Resources) error {
	if r.CpusetCpus != "" {
		if err := cgroups.WriteFile(path, "cpuset.cpus", r.CpusetCpus); err != nil {
			return err
		}
	}
	if r.CpusetMems != "" {
		if err := cgroups.WriteFile(path, "cpuset.mems", r.CpusetMems); err != nil {
			return err
		}
	}
	return nil
}

func (s *CpusetSubsystem) ApplyDir(dir string, r *config.Resources, pid int) error {
	// 如果未挂载 cpuset cgroup，这种情况可能会发生。
	// 只需不做任何操作，也不要让系统失败。
	if dir == "" {
		return nil
	}
	// 'ensureParent' 以父级开始，因为我们不希望显式继承自父级，这可能与'cpuset.cpu_exclusive'发生冲突。
	if err := cpusetEnsureParent(filepath.Dir(dir)); err != nil {
		return err
	}
	if err := os.Mkdir(dir, 0o755); err != nil && !os.IsExist(err) {
		return err
	}
	// We didn't inherit cpuset configs from parent, but we have
	// to ensure cpuset configs are set before moving task into the
	// cgroup.
	// The logic is, if user specified cpuset configs, use these
	// specified configs, otherwise, inherit from parent. This makes
	// cpuset configs work correctly with 'cpuset.cpu_exclusive', and
	// keep backward compatibility.
	if err := s.ensureCpusAndMems(dir, r); err != nil {
		return err
	}
	// Since we are not using apply(), we need to place the pid
	// into the procs file.
	return cgroups.WriteCgroupProc(dir, pid)
}

func cpusetEnsureParent(current string) error {
	var st unix.Statfs_t

	parent := filepath.Dir(current)
	err := unix.Statfs(parent, &st)
	if err == nil && st.Type != unix.CGROUP_SUPER_MAGIC {
		return nil
	}
	// Treat non-existing directory as cgroupfs as it will be created,
	// and the root cpuset directory obviously exists.
	if err != nil && err != unix.ENOENT {
		return &os.PathError{Op: "statfs", Path: parent, Err: err}
	}

	if err := cpusetEnsureParent(parent); err != nil {
		return err
	}
	if err := os.Mkdir(current, 0o755); err != nil && !os.IsExist(err) {
		return err
	}
	return cpusetCopyIfNeeded(current, parent)
}

// cpusetCopyIfNeeded 函数会将父目录中的 cpuset.cpus 和 cpuset.mems 文件内容复制到当前目录，前提是这些文件的内容为0。
func cpusetCopyIfNeeded(current, parent string) error {
	currentCpus, currentMems, err := getCpusetSubsystemSettings(current)
	if err != nil {
		return err
	}
	parentCpus, parentMems, err := getCpusetSubsystemSettings(parent)
	if err != nil {
		return err
	}

	if isEmptyCpuset(currentCpus) {
		if err := cgroups.WriteFile(current, "cpuset.cpus", parentCpus); err != nil {
			return err
		}
	}
	if isEmptyCpuset(currentMems) {
		if err := cgroups.WriteFile(current, "cpuset.mems", parentMems); err != nil {
			return err
		}
	}
	return nil
}
func isEmptyCpuset(str string) bool {
	return str == "" || str == "\n"
}
func (s *CpusetSubsystem) ensureCpusAndMems(path string, r *config.Resources) error {
	if err := s.Set(path, r); err != nil {
		return err
	}
	return cpusetCopyIfNeeded(path, filepath.Dir(path))
}

func getCpusetSubsystemSettings(parent string) (cpus, mems string, err error) {
	if cpus, err = cgroups.ReadFile(parent, "cpuset.cpus"); err != nil {
		return
	}
	if mems, err = cgroups.ReadFile(parent, "cpuset.mems"); err != nil {
		return
	}
	return cpus, mems, nil
}
