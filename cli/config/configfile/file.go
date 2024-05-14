package configfile

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

// ConfigFile ~/.sudocker/config.json 文件信息
type ConfigFile struct {
	PsFormat             string            `json:"psFormat,omitempty"`
	ImagesFormat         string            `json:"imagesFormat,omitempty"`
	NetworksFormat       string            `json:"networksFormat,omitempty"`
	PluginsFormat        string            `json:"pluginsFormat,omitempty"`
	VolumesFormat        string            `json:"volumesFormat,omitempty"`
	StatsFormat          string            `json:"statsFormat,omitempty"`
	DetachKeys           string            `json:"detachKeys,omitempty"`
	Filename             string            `json:"-"` // Note: for internal use only
	ServiceInspectFormat string            `json:"serviceInspectFormat,omitempty"`
	ServicesFormat       string            `json:"servicesFormat,omitempty"`
	TasksFormat          string            `json:"tasksFormat,omitempty"`
	SecretFormat         string            `json:"secretFormat,omitempty"`
	ConfigFormat         string            `json:"configFormat,omitempty"`
	NodesFormat          string            `json:"nodesFormat,omitempty"`
	CurrentContext       string            `json:"currentContext,omitempty"`
	Aliases              map[string]string `json:"aliases,omitempty"`
	Features             map[string]string `json:"features,omitempty"`
}

// 给定文件名 'fn'，初始化配置文件
func New(fn string) *ConfigFile {
	return &ConfigFile{
		Filename: fn,
		Aliases:  make(map[string]string),
	}
}

// LoadFromReader读取配置文件
func (configFile *ConfigFile) LoadFromReader(configData io.Reader) error {
	if err := json.NewDecoder(configData).Decode(configFile); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

// GetFilename 返回配置文件名
func (configFile *ConfigFile) GetFilename() string {
	return configFile.Filename
}
