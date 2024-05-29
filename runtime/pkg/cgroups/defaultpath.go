package cgroups

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/DeJeune/sudocker/runtime/utils"
)

const UnifiedMountpoint = "/sys/fs/cgroup"

func defaultDirPath(c *config.Cgroup) (string, error) {
	if (c.Name != "" || c.Parent != "") && c.Path != "" {
		return "", fmt.Errorf("cgroup: either Path or Name and Parent should be used, got %+v", c)
	}

	return _defaultDirPath(UnifiedMountpoint, c.Path, c.Parent, c.Name)
}

func _defaultDirPath(root, cgPath, cgParent, cgName string) (string, error) {
	if (cgName != "" || cgParent != "") && cgPath != "" {
		return "", errors.New("cgroup: either Path or Name and Parent should be used")
	}

	innerPath := utils.CleanPath(cgPath)
	if innerPath == "" {
		cgParent := utils.CleanPath(cgParent)
		cgName := utils.CleanPath(cgName)
		innerPath = filepath.Join(cgParent, cgName)
	}
	if filepath.IsAbs(innerPath) {
		return filepath.Join(root, innerPath), nil
	}

	ownCgroup, err := parseCgroupFile("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	ownCgroup = filepath.Dir(ownCgroup)

	return filepath.Join(root, ownCgroup, innerPath), nil
}

func parseCgroupFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return parseCgroupFromReader(f)
}

func parseCgroupFromReader(r io.Reader) (string, error) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		var (
			text  = s.Text()
			parts = strings.SplitN(text, ":", 3)
		)
		if len(parts) < 3 {
			return "", fmt.Errorf("invalid cgroup entry: %q", text)
		}
		// text is like "0::/user.slice/user-1001.slice/session-1.scope"
		if parts[0] == "0" && parts[1] == "" {
			return parts[2], nil
		}
	}
	if err := s.Err(); err != nil {
		return "", err
	}
	return "", errors.New("cgroup path not found")
}
