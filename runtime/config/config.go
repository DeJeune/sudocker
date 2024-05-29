package config

import (
	"github.com/DeJeune/sudocker/runtime/pkg/devices"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type (
	HookName string
	HookList []Hook
	Hooks    map[HookName]HookList
)

type Hook interface {
	// Run executes the hook with the provided state.
	Run(*specs.State) error
}

type Capabilities struct {
	// Bounding is the set of capabilities checked by the kernel.
	Bounding []string
	// Effective is the set of capabilities checked by the kernel.
	Effective []string
	// Inheritable is the capabilities preserved across execve.
	Inheritable []string
	// Permitted is the limiting superset for effective capabilities.
	Permitted []string
	// Ambient is the ambient set of capabilities that are kept.
	Ambient []string
}

type Config struct {
	// NoPivotRoot will use MS_MOVE and a chroot to jail the process into the container's rootfs
	// This is a common option when the container is running in ramdisk
	NoPivotRoot bool `json:"no_pivot_root"`

	// ParentDeathSignal specifies the signal that is sent to the container's process in the case
	// that the parent process dies.
	ParentDeathSignal int `json:"parent_death_signal"`

	// Path to a directory containing the container's root filesystem.
	Rootfs string `json:"rootfs"`

	// Umask is the umask to use inside of the container.
	Umask *uint32 `json:"umask"`

	// Readonlyfs will remount the container's rootfs as readonly where only externally mounted
	// bind mounts are writtable.
	Readonlyfs bool `json:"readonlyfs"`

	// Specifies the mount propagation flags to be applied to /.
	RootPropagation int `json:"rootPropagation"`

	// Mounts specify additional source and destination paths that will be mounted inside the container's
	// rootfs and mount namespace if specified
	Mounts []*Mount `json:"mounts"`

	// The device nodes that should be automatically created within the container upon container start.  Note, make sure that the node is marked as allowed in the cgroup as well!
	Devices []*devices.Device `json:"devices"`

	MountLabel string `json:"mount_label"`

	// Hostname optionally sets the container's hostname if provided
	Hostname string `json:"hostname"`

	// Domainname optionally sets the container's domainname if provided
	Domainname string `json:"domainname"`

	// Namespaces specifies the container's namespaces that it should setup when cloning the init process
	// If a namespace is not provided that namespace is shared from the container's parent process
	Namespaces Namespaces `json:"namespaces"`

	// Capabilities specify the capabilities to keep when executing the process inside the container
	// All capabilities not specified will be dropped from the processes capability mask
	Capabilities *Capabilities `json:"capabilities"`

	// Networks specifies the container's network setup to be created
	Networks []*Network `json:"networks"`

	// Routes can be specified to create entries in the route table as the container is started
	// Routes []*Route `json:"routes"`

	// Cgroups specifies specific cgroup settings for the various subsystems that the container is
	// placed into to limit the resources the container has available
	Cgroups *Cgroup `json:"cgroups"`

	// AppArmorProfile specifies the profile to apply to the process running in the container and is
	// change at the time the process is execed
	// AppArmorProfile string `json:"apparmor_profile,omitempty"`

	// ProcessLabel specifies the label to apply to the process running in the container.  It is
	// commonly used by selinux
	// ProcessLabel string `json:"process_label,omitempty"`

	// Rlimits specifies the resource limits, such as max open files, to set in the container
	// If Rlimits are not set, the container will inherit rlimits from the parent process
	// Rlimits []Rlimit `json:"rlimits,omitempty"`

	// OomScoreAdj specifies the adjustment to be made by the kernel when calculating oom scores
	// for a process. Valid values are between the range [-1000, '1000'], where processes with
	// higher scores are preferred for being killed. If it is unset then we don't touch the current
	// value.
	// More information about kernel oom score calculation here: https://lwn.net/Articles/317814/
	// OomScoreAdj *int `json:"oom_score_adj,omitempty"`

	// UIDMappings is an array of User ID mappings for User Namespaces
	UIDMappings []IDMap `json:"uid_mappings"`

	// GIDMappings is an array of Group ID mappings for User Namespaces
	GIDMappings []IDMap `json:"gid_mappings"`

	// MaskPaths specifies paths within the container's rootfs to mask over with a bind
	// mount pointing to /dev/null as to prevent reads of the file.
	MaskPaths []string `json:"mask_paths"`

	// ReadonlyPaths specifies paths within the container's rootfs to remount as read-only
	// so that these files prevent any writes.
	ReadonlyPaths []string `json:"readonly_paths"`

	// Sysctl is a map of properties and their values. It is the equivalent of using
	// sysctl -w my.property.name value in Linux.
	Sysctl map[string]string `json:"sysctl"`

	// Seccomp allows actions to be taken whenever a syscall is made within the container.
	// A number of rules are given, each having an action to be taken if a syscall matches it.
	// A default action to be taken if no rules match is also given.
	// Seccomp *Seccomp `json:"seccomp"`

	// NoNewPrivileges controls whether processes in the container can gain additional privileges.
	// NoNewPrivileges bool `json:"no_new_privileges,omitempty"`

	// Hooks are a collection of actions to perform at various container lifecycle events.
	// CommandHooks are serialized to JSON, but other hooks are not.
	Hooks Hooks

	// Version is the version of opencontainer specification that is supported.
	Version string `json:"version"`

	// Labels are user defined metadata that is stored in the config and populated on the state
	Labels []string `json:"labels"`

	// NoNewKeyring will not allocated a new session keyring for the container.  It will use the
	// callers keyring in this case.
	NoNewKeyring bool `json:"no_new_keyring"`

	// RootlessEUID is set when the runc was launched with non-zero EUID.
	// Note that RootlessEUID is set to false when launched with EUID=0 in userns.
	// When RootlessEUID is set, runc creates a new userns for the container.
	// (config.json needs to contain userns settings)
	// RootlessEUID bool `json:"rootless_euid,omitempty"`

	// RootlessCgroups is set when unlikely to have the full access to cgroups.
	// When RootlessCgroups is set, cgroups errors are ignored.
	// RootlessCgroups bool `json:"rootless_cgroups,omitempty"`

	// TimeOffsets specifies the offset for supporting time namespaces.
	// TimeOffsets map[string]specs.LinuxTimeOffset `json:"time_offsets,omitempty"`

	// Scheduler represents the scheduling attributes for a process.
	// Scheduler *Scheduler `json:"scheduler,omitempty"`

	// Personality contains configuration for the Linux personality syscall.
	// Personality *LinuxPersonality `json:"personality,omitempty"`

	// IOPriority is the container's I/O priority.
	// IOPriority *IOPriority `json:"io_priority,omitempty"`
}

// IDMap represents UID/GID Mappings for User Namespaces.
type IDMap struct {
	ContainerID int64 `json:"container_id"`
	HostID      int64 `json:"host_id"`
	Size        int64 `json:"size"`
}
