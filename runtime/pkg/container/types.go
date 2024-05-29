package container

import (
	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/DeJeune/sudocker/runtime/pkg/network"
)

type Rootfs struct {
	ContainerDir string `json:"ContainerDir"`
	ImageDir     string `json:"ImageDir"`
	WriteDir     string `json:"WriteDir"`
	MergeDir     string `json:"MergeDir"`
}

type Container struct {
	Detach        bool                `json:"Detach"`
	Uuid          string              `json:"Uuid"`
	Name          string              `json:"Name"`
	Hostname      string              `json:"Hostname"`
	Dns           []string            `json:"Dns"`
	Image         string              `json:"Image"`
	CreateTime    string              `json:"CreateTime"`
	Status        string              `json:"Status"`
	StorageDriver string              `json:"StorageDriver"`
	Rootfs        *Rootfs             `json:"Rootfs"`
	Commands      []string            `json:"Commands"`
	Cgroups       *config.Cgroup      `json:"Cgroup"`
	Volumes       map[string]string   `json:"Volumes"`
	Envs          map[string]string   `json:"Envs"`
	Ports         map[string]string   `json:"Ports"`
	Endpoints     []*network.Endpoint `json:"Endpoints"`
}

type Driver interface {
	Name() string
	Allowed() bool
	MountRootfs(*Container) error
	MountVolume(*Container) error
}
