package config

import "fmt"

type BlockIODevice struct {
	// Major是设备的主要号
	Major int64 `json:"major"`
	// Minor是设备的次要号
	Minor int64 `json:"minor"`
}

type WeightDevice struct {
	BlockIODevice

	// 设备的带宽,10-1000
	Weight uint16 `json:"weight"`
	// 用于决定给定 cgroup 中任务在与 cgroup 的子 cgroup 竞争时
	// 具有多少权重的 weight 的等效项。
	LeafWeight uint16 `json:"leafWeight"`
}

// 返回一个WeightDevice配置对象的指针
func NewWeightDevice(major, minor int64, weight, leafWeight uint16) *WeightDevice {
	wd := &WeightDevice{}
	wd.Major = major
	wd.Minor = minor
	wd.Weight = weight
	wd.LeafWeight = leafWeight
	return wd
}

// 格式化字符串为了写入cgroup文件
func (wd *WeightDevice) WeightString() string {
	return fmt.Sprintf("%d:%d %d", wd.Major, wd.Minor, wd.Weight)
}

// 格式化字符串为了写入cgroup文件
func (wd *WeightDevice) LeafWeightString() string {
	return fmt.Sprintf("%d:%d %d", wd.Major, wd.Minor, wd.LeafWeight)
}

// 限制速率的设备:`major:minor`
type ThrottleDevice struct {
	BlockIODevice
	Rate uint64 `json:"rate"`
}

func NewThrottleDevice(major, minor int64, rate uint64) *ThrottleDevice {
	td := &ThrottleDevice{}
	td.Major = major
	td.Minor = minor
	td.Rate = rate
	return td
}

func (td *ThrottleDevice) String() string {
	return fmt.Sprintf("%d:%d %d", td.Major, td.Minor, td.Rate)
}

func (td *ThrottleDevice) StringName(name string) string {
	return fmt.Sprintf("%d:%d %s=%d", td.Major, td.Minor, name, td.Rate)
}
