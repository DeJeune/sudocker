package config

import "golang.org/x/sys/unix"

const (
	// EXT_COPYUP 是一个指令，用于在将 tmpfs 挂载到某个目录上时，复制该目录的内容。
	EXT_COPYUP = 1 << iota
)

// Type represents the type of a mount.
type Type string

// Type constants
const (
	// TypeBind is the type for mounting host dir
	TypeBind Type = "bind"
	// TypeVolume is the type for remote storage volumes
	TypeVolume Type = "volume"
	// TypeTmpfs is the type for mounting tmpfs
	TypeTmpfs Type = "tmpfs"
	// TypeNamedPipe is the type for mounting Windows named pipes
	TypeNamedPipe Type = "npipe"
	// TypeCluster is the type for Swarm Cluster Volumes.
	TypeCluster Type = "cluster"
)

type MountIDMapping struct {
	// Recursive indicates if the mapping needs to be recursive.
	Recursive bool `json:"recursive"`

	// UserNSPath is a path to a user namespace that indicates the necessary
	// id-mappings for MOUNT_ATTR_IDMAP. If set to non-"", UIDMappings and
	// GIDMappings must be set to nil.
	UserNSPath string `json:"userns_path,omitempty"`

	// UIDMappings is the uid mapping set for this mount, to be used with
	// MOUNT_ATTR_IDMAP.
	UIDMappings []IDMap `json:"uid_mappings,omitempty"`

	// GIDMappings is the gid mapping set for this mount, to be used with
	// MOUNT_ATTR_IDMAP.
	GIDMappings []IDMap `json:"gid_mappings,omitempty"`
}

type Mount struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Device      string `json:"device"`
	Fstype      string `json:"fstype"`
	Flags       int    `json:"flags"`
	// Mount flags that were explicitly cleared in the configuration (meaning
	// the user explicitly requested that these flags *not* be set).
	ClearedFlags int `json:"cleared_flags"`
	// Propagation Flags
	PropagationFlags []int  `json:"propagation_flags"`
	Data             string `json:"data"`
	Relabel          string `json:"relabel"`
	// RecAttr represents mount properties to be applied recursively (AT_RECURSIVE), see mount_setattr(2).
	RecAttr *unix.MountAttr `json:"rec_attr"`

	// Extensions are additional flags that are specific to runc.
	Extensions int             `json:"extensions"`
	IDMapping  *MountIDMapping `json:"id_mapping,omitempty"`
}

func (m *Mount) IsBind() bool {
	return m.Flags&unix.MS_BIND != 0
}

func (m *Mount) IsIDMapped() bool {
	return m.IDMapping != nil
}
