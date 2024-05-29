package container

type ContainerState string

const (
	// StateCreating:容器处于创建中
	StateCreating ContainerState = "creating"

	// StateCreated：容器已经创建
	StateCreated ContainerState = "created"

	// StateRunning：容器执行了一个用户空间进程，且未退出
	StateRunning ContainerState = "running"

	// StateStopped：容器已经退出
	StateStopped ContainerState = "stopped"
)

type State struct {
	// Version is the version of the specification that is supported.
	Version string `json:"ociVersion"`
	// ID is the container ID
	ID string `json:"id"`
	// Status is the runtime status of the container.
	Status ContainerState `json:"status"`
	// Pid is the process ID for the container process.
	Pid int `json:"pid,omitempty"`
	// Bundle is the path to the container's bundle directory.
	Bundle string `json:"bundle"`
	// Annotations are key values associated with the container.
	Annotations map[string]string `json:"annotations,omitempty"`
}

type ContainerProcessState struct {
	// Version is the version of the specification that is supported.
	Version string `json:"ociVersion"`
	// Fds is a string array containing the names of the file descriptors passed.
	// The index of the name in this array corresponds to index of the file
	// descriptor in the `SCM_RIGHTS` array.
	Fds []string `json:"fds"`
	// Pid is the process ID as seen by the runtime.
	Pid int `json:"pid"`
	// Opaque metadata.
	Metadata string `json:"metadata,omitempty"`
	// State of the container.
	State State `json:"state"`
}
