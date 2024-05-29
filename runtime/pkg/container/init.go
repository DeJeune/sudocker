package container

import (
	"github.com/DeJeune/sudocker/runtime/config"
)

// 根据参数和环境变量
func (c *Container) NewParentProcess(p *Process) error {

}

type initConfig struct {
	Args        []string          `json:"args"`
	Env         []string          `json:"env"`
	User        string            `json:"user"`
	Networks    []*config.Network `json:"network"`
	ContainerID string            `json:"containerid"`
	CgroupPath  string            `json:"cgroup_path,omitempty"`
	Config      *config.Config    `json:"config"`
	State       *State            `json:"state,omitempty"`
}
