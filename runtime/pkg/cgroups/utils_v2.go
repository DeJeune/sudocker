package cgroups

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DeJeune/sudocker/runtime/pkg/userns"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	CgroupProcesses   = "cgroup.procs"
	unifiedMountpoint = "/sys/fs/cgroup"
	hybridMountpoint  = "/sys/fs/cgroup/unified"
)

var (
	isUnifiedOnce sync.Once
	isUnified     bool
	isHybridOnce  sync.Once
	isHybrid      bool
)

func IsCgroup2UnifiedMode() bool {
	isUnifiedOnce.Do(func() {
		var st unix.Statfs_t
		err := unix.Statfs(unifiedMountpoint, &st)
		if err != nil {
			level := logrus.WarnLevel
			if os.IsNotExist(err) && userns.RunningInUserNS() {
				// For rootless containers, sweep it under the rug.
				level = logrus.DebugLevel
			}
			logrus.StandardLogger().Logf(level,
				"statfs %s: %v; assuming cgroup v1", unifiedMountpoint, err)
		}
		isUnified = st.Type == unix.CGROUP2_SUPER_MAGIC
	})
	return isUnified
}

type Mount struct {
	Mountpoint string
	Root       string
	Subsystems []string
}

func PathExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

// 写入pid到cgroups的cgroup.proc文件
func WriteCgroupProc(dir string, pid int) error {
	if dir == "" {
		return fmt.Errorf("no such directory for %s", CgroupProcesses)
	}
	// don't attach any pid to the cgroup
	if pid == -1 {
		return nil
	}

	file, err := OpenFile(dir, CgroupProcesses, os.O_WRONLY)
	if err != nil {
		return fmt.Errorf("failed to write %v:%w", pid, err)
	}
	defer file.Close()

	for i := 0; i < 5; i++ {
		_, err = file.WriteString(strconv.Itoa(pid))
		if err == nil {
			return nil
		}
		// 可能写入任务还没有创建完全
		if errors.Is(err, unix.EINVAL) {
			time.Sleep(30 * time.Millisecond)
			continue
		}
	}
	return fmt.Errorf("failed to write %v: %w", pid, err)
}

// ParseCgroupFile parses the given cgroup file, typically /proc/self/cgroup
// or /proc/<pid>/cgroup, into a map of subsystems to cgroup paths, e.g.
//
//	"cpu": "/user.slice/user-1000.slice"
//	"pids": "/user.slice/user-1000.slice"
//
// etc.
//
// Note that for cgroup v2 unified hierarchy, there are no per-controller
// cgroup paths, so the resulting map will have a single element where the key
// is empty string ("") and the value is the cgroup path the <pid> is in.
func ParseCgroupFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseCgroupFromReader(f)
}

// helper function for ParseCgroupFile to make testing easier
func parseCgroupFromReader(r io.Reader) (map[string]string, error) {
	s := bufio.NewScanner(r)
	cgroups := make(map[string]string)

	for s.Scan() {
		text := s.Text()
		// from cgroups(7):
		// /proc/[pid]/cgroup
		// ...
		// For each cgroup hierarchy ... there is one entry
		// containing three colon-separated fields of the form:
		//     hierarchy-ID:subsystem-list:cgroup-path
		parts := strings.SplitN(text, ":", 3)
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid cgroup entry: must contain at least two colons: %v", text)
		}

		for _, subs := range strings.Split(parts[1], ",") {
			cgroups[subs] = parts[2]
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	return cgroups, nil
}

func GetCgroupMounts(all bool) ([]Mount, error) {
	if IsCgroup2UnifiedMode() {
		// TODO: remove cgroupv2 case once all external users are converted
		availableControllers, err := GetAllSubsystems()
		if err != nil {
			return nil, err
		}
		m := Mount{
			Mountpoint: unifiedMountpoint,
			Root:       unifiedMountpoint,
			Subsystems: availableControllers,
		}
		return []Mount{m}, nil
	}

	return getCgroupMountsV1(all)
}

func GetAllSubsystems() ([]string, error) {
	// /proc/cgroups is meaningless for v2
	// https://github.com/torvalds/linux/blob/v5.3/Documentation/admin-guide/cgroup-v2.rst#deprecated-v1-core-features
	if IsCgroup2UnifiedMode() {
		// "pseudo" controllers do not appear in /sys/fs/cgroup/cgroup.controllers.
		// - devices: implemented in kernel 4.15
		// - freezer: implemented in kernel 5.2
		// We assume these are always available, as it is hard to detect availability.
		pseudo := []string{"devices", "freezer"}
		data, err := ReadFile("/sys/fs/cgroup", "cgroup.controllers")
		if err != nil {
			return nil, err
		}
		subsystems := append(pseudo, strings.Fields(data)...)
		return subsystems, nil
	}
	f, err := os.Open("/proc/cgroups")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	subsystems := []string{}

	s := bufio.NewScanner(f)
	for s.Scan() {
		text := s.Text()
		if text[0] != '#' {
			parts := strings.Fields(text)
			if len(parts) >= 4 && parts[3] != "0" {
				subsystems = append(subsystems, parts[0])
			}
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return subsystems, nil
}

// rmdir tries to remove a directory, optionally retrying on EBUSY.
func rmdir(path string, retry bool) error {
	delay := time.Millisecond
	tries := 10

again:
	err := unix.Rmdir(path)
	switch err { // nolint:errorlint // unix errors are bare
	case nil, unix.ENOENT:
		return nil
	case unix.EINTR:
		goto again
	case unix.EBUSY:
		if retry && tries > 0 {
			time.Sleep(delay)
			delay *= 2
			tries--
			goto again

		}
	}
	return &os.PathError{Op: "rmdir", Path: path, Err: err}
}

// RemovePath aims to remove cgroup path. It does so recursively,
// by removing any subdirectories (sub-cgroups) first.
func RemovePath(path string) error {
	// Try the fast path first.
	if err := rmdir(path, false); err == nil {
		return nil
	}

	infos, err := os.ReadDir(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, info := range infos {
		if info.IsDir() {
			// We should remove subcgroup first.
			if err = RemovePath(filepath.Join(path, info.Name())); err != nil {
				break
			}
		}
	}
	if err == nil {
		err = rmdir(path, true)
	}
	return err
}

// RemovePaths iterates over the provided paths removing them.
func RemovePaths(paths map[string]string) (err error) {
	for s, p := range paths {
		if err := RemovePath(p); err == nil {
			delete(paths, s)
		}
	}
	if len(paths) == 0 {
		//nolint:ineffassign,staticcheck // done to help garbage collecting: opencontainers/runc#2506
		// TODO: switch to clear once Go < 1.21 is not supported.
		paths = make(map[string]string)
		return nil
	}
	return fmt.Errorf("Failed to remove paths: %v", paths)
}

// 将cgroup v1的BlkIOWeight转为cgroup v2 的IOWeight
func ConvertBlkIOToIOWeightValue(blkIoWeight uint16) uint64 {
	if blkIoWeight == 0 {
		return 0
	}
	return 1 + (uint64(blkIoWeight)-10)*9999/990
}

// 转换cgroup v1的memory swap 到cgrop v2的 swap
func ConvertMemorySwapToCgroupV2Value(memorySwap, memory int64) (int64, error) {

	if memory == -1 && memorySwap == 0 {
		return -1, nil
	}
	if memorySwap == -1 || memorySwap == 0 {
		// -1 is "max", 0 is "unset", so treat as is
		return memorySwap, nil
	}
	// sanity checks
	if memory == 0 || memory == -1 {
		return 0, errors.New("unable to set swap limit without memory limit")
	}
	if memory < 0 {
		return 0, fmt.Errorf("invalid memory value: %d", memory)
	}
	if memorySwap < memory {
		return 0, errors.New("memory+swap limit should be >= memory limit")
	}

	return memorySwap - memory, nil
}

func ConvertCPUSharesToCgroupV2Value(cpuShares uint64) uint64 {
	if cpuShares == 0 {
		return 0
	}
	return (1 + ((cpuShares-2)*9999)/262142)
}
