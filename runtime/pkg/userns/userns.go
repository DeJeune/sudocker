package userns

import (
	"bufio"
	"fmt"
	"os"
	"sync"
)

var (
	inUserNS bool
	nsOnce   sync.Once
)

var RunningInUserNS = runningInUserNS

func runningInUserNS() bool {
	nsOnce.Do(func() {
		file, err := os.Open("/proc/self/uid_map")
		if err != nil {
			// This kernel-provided file only exists if user namespaces are supported.
			return
		}
		defer file.Close()

		buf := bufio.NewReader(file)
		l, _, err := buf.ReadLine()
		if err != nil {
			return
		}

		inUserNS = uidMapInUserNS(string(l))
	})
	return inUserNS
}

// 在 uidMapInUserNS 函数中：

// 如果 uidMap 为空，表示文件存在但内容为空，这通常是用户命名空间刚创建时的状态，返回 true。
// 使用 fmt.Sscanf 解析 uidMap 的内容为三个整数 a, b, c。如果解析失败，假设不在用户命名空间中，返回 false。
// 根据 user_namespaces(7) 的描述，初始用户命名空间的 /proc/self/uid_map 显示为 0 0 4294967295。
// 如果解析结果符合这个模式，表示处于初始命名空间，返回 false；否则返回 true。
func uidMapInUserNS(uidMap string) bool {
	if uidMap == "" {
		// File exist but empty (the initial state when userns is created,
		// see user_namespaces(7)).
		return true
	}

	var a, b, c int64
	if _, err := fmt.Sscanf(uidMap, "%d %d %d", &a, &b, &c); err != nil {
		// Assume we are in a regular, non user namespace.
		return false
	}

	// As per user_namespaces(7), /proc/self/uid_map of
	// the initial user namespace shows 0 0 4294967295.
	initNS := a == 0 && b == 0 && c == 4294967295
	return !initNS
}
