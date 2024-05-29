package cgroups

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DeJeune/sudocker/runtime/pkg/userns"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
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

const (
	CgroupProcesses   = "cgroup.procs"
	unifiedMountpoint = "/sys/fs/cgroup"
	hybridMountpoint  = "/sys/fs/cgroup/unified"
)

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

// numToStr转换一个int64类型变量为字符串类型变量
// 然后写到一个cgroupv2文件，文件名以.min, .max, .low, .high结尾
func numToStr(value int64) (ret string) {
	switch {
	case value == 0:
		ret = ""
	default:
		ret = strconv.FormatInt(value, 10)
	}

	return ret
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

// 记录解析错误
type ParseError struct {
	Path string
	File string
	Err  error
}

func (e *ParseError) Error() string {
	return "unable to parse " + path.Join(e.Path, e.File) + ": " + e.Err.Error()
}

func (e *ParseError) UnWrap() error {
	return e.Err
}

// 转换字符串为64位无符号整数
func ParseUint(s string, base, bitSize int) (uint64, error) {
	value, err := strconv.ParseUint(s, base, bitSize)
	if err != nil {
		intValue, intErr := strconv.ParseInt(s, base, bitSize)
		// 1. Handle negative values greater than MinInt64 (and)
		// 2. Handle negative values lesser than MinInt64
		if intErr == nil && intValue < 0 {
			return 0, nil
		} else if errors.Is(intErr, strconv.ErrRange) && intValue < 0 {
			return 0, nil
		}

		return value, err
	}

	return value, nil
}

// 解析cgroup里的"key value"
// 例如,"io_service_bytes_1234" 将要返回"io_service_bytes", 1234
func ParseKeyValue(t string) (string, uint64, error) {
	parts := strings.SplitN(t, " ", 3)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("line %q is not in key value format", t)
	}

	value, err := ParseUint(parts[1], 10, 64)
	if err != nil {
		return "", 0, err
	}

	return parts[0], value, nil
}

func GetCgroupParamString(path, file string) (string, error) {
	contents, err := ReadFile(path, file)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(contents), nil
}

func GetCgroupParamUint(path, file string) (uint64, error) {
	contents, err := GetCgroupParamString(path, file)
	if err != nil {
		return 0, err
	}
	contents = strings.TrimSpace(contents)
	if contents == "max" {
		return math.MaxUint64, nil
	}

	res, err := ParseUint(contents, 10, 64)
	if err != nil {
		return res, &ParseError{Path: path, File: file, Err: err}
	}
	return res, nil
}
