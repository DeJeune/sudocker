package config

type Cgroup struct {
	// cgroup名称
	Name string `json:"name,omitempty"`
	// cgroup/slice 的parent名称
	Parent string `json:"parent,omitempty"`
	// cgroup被创建，或者被容器添加的路径
	Path string `json:"path"`
	*Resources
	ScopePrefix string `json:"scope_prefix"`
	Rootless    bool
}

type Resources struct {
	Unified                      map[string]string `json:"unified"`
	CpuIdle                      *int64            `json:"cpu_idle,omitempty"`
	CpuShares                    uint64            `json:"cpu_shares"`
	CpuQuota                     int64             `json:"cpu_quota"`
	CpuBurst                     *uint64           `json:"cpu_burst"`
	CpuPeriod                    uint64            `json:"cpu_period"`
	CpuRtRuntime                 int64             `json:"cpu_rt_quota"`
	CpuRtPeriod                  uint64            `json:"cpu_rt_period"`
	CpusetCpus                   string            `json:"cpuset_cpus"`
	CpusetMems                   string            `json:"cpuset_mems"`
	CpuWeight                    uint64            `json:"cpu_weight"`
	Memory                       int64             `json:"memory"`
	MemoryReservation            int64             `json:"memory_reservation"`
	MemorySwap                   int64             `json:"memory_swap"`
	MemorySwappiness             *uint64           `json:"memory_swappiness"`
	IoMax                        string            `json:"IoMax"`
	IoWeight                     string            `json:"IoWeight"`
	IoStat                       string            `json:"IoStat"`
	IoPressure                   string            `json:"IoPressure"`
	PidsLimit                    int64             `json:"pids_limit"`
	BlkioWeight                  uint16            `json:"blkio_weight"`
	BlkioLeafWeight              uint16            `json:"blkio_leaf_weight"`
	BlkioWeightDevice            []*WeightDevice   `json:"blkio_weight_device"`
	BlkioThrottleReadBpsDevice   []*ThrottleDevice `json:"blkio_throttle_read_bps_device"`
	BlkioThrottleWriteBpsDevice  []*ThrottleDevice `json:"blkio_throttle_write_bps_device"`
	BlkioThrottleReadIOPSDevice  []*ThrottleDevice `json:"blkio_throttle_read_iops_device"`
	BlkioThrottleWriteIOPSDevice []*ThrottleDevice `json:"blkio_throttle_write_iops_device"`
	MemoryCheckBeforeUpdate      bool              `json:"memory_check_before_update"`

	SkipDevices bool `json:"-"`

	HugetlbLimit []*HugepageLimit `json:"hugetlb_limit"`
}
