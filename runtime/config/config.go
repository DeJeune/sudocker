package config

import (
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
)

type (
	HookName string
	HookList []Hook
	Hooks    map[HookName]HookList
)

const (
	// Prestart commands are executed after the container namespaces are created,
	// but before the user supplied command is executed from init.
	// Note: This hook is now deprecated
	// Prestart commands are called in the Runtime namespace.
	Prestart HookName = "prestart"

	// CreateRuntime commands 必须作为create操作的一部分在运行时环境被创建之后，容器进程被更改根文件系统之后调用
	// CreateRuntime is called immediately after the deprecated Prestart hook.
	// CreateRuntime commands are called in the Runtime Namespace.
	CreateRuntime HookName = "createRuntime"

	// CreateContainer commands MUST be called as part of the create operation after
	// the runtime environment has been created but before the pivot_root has been executed.
	// CreateContainer commands are called in the Container namespace.
	CreateContainer HookName = "createContainer"

	// StartContainer commands MUST be called as part of the start operation and before
	// the container process is started.
	// StartContainer commands are called in the Container namespace.
	StartContainer HookName = "startContainer"

	// Poststart commands are executed after the container init process starts.
	// Poststart commands are called in the Runtime Namespace.
	Poststart HookName = "poststart"

	// Poststop commands are executed after the container init process exits.
	// Poststop commands are called in the Runtime Namespace.
	Poststop HookName = "poststop"
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
	OpenStdin    bool
	StdinOnce    bool
	AttachStdin  bool
	AttachStdout bool
	AttachStderr bool
	Tty          bool
	Cmd          []string
	Image        string
}

type HostConfig struct {
	Binds []string // List of volume bindings for this container
	*Resources
}

// IDMap represents UID/GID Mappings for User Namespaces.
type IDMap struct {
	ContainerID int64 `json:"container_id"`
	HostID      int64 `json:"host_id"`
	Size        int64 `json:"size"`
}

func (hooks Hooks) Run(name HookName, state *specs.State) error {
	list := hooks[name]
	for i, h := range list {
		if err := h.Run(state); err != nil {
			return fmt.Errorf("error running %s hook #%d: %w", name, i, err)
		}
	}
	return nil
}
