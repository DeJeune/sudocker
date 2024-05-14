package config

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	configfile "github.com/DeJeune/sudocker/cli/config/configfile"
	"github.com/pkg/errors"
)

const (
	EnvOverrideConfigDir = "SUDOCKER_CONFIG"
	ConfigFileName       = "config.json"
	configFileDir        = ".sudocker"
)

var (
	configDir     string
	initConfigDir = new(sync.Once)
)

func resetConfigDir() {
	configDir = ""
	initConfigDir = new(sync.Once)
}

// 配置文件存储的位置
func Dir() string {
	initConfigDir.Do(func() {
		configDir = os.Getenv(EnvOverrideConfigDir)
		if configDir == "" {
			u, err := user.Current()
			if err != nil {
				log.Fatal(err)
			}
			configDir = filepath.Join(u.HomeDir, configFileDir)
		}
	})
	return configDir
}

func SetDir(dir string) {
	initConfigDir.Do(func() {})
	configDir = filepath.Clean(dir)
}

func Path(p ...string) (string, error) {
	path := filepath.Join(append([]string{Dir()}, p...)...)
	if !strings.HasPrefix(path, Dir()+string(filepath.Separator)) {
		return "", errors.Errorf("path %q is outside of root config director %q", path, Dir())
	}
	return path, nil
}

// LoadFromReader 从Reader创建配置文件对象
func LoadFromReader(configData io.Reader) (*configfile.ConfigFile, error) {
	configFile := configfile.ConfigFile{}
	err := configFile.LoadFromReader(configData)
	return &configFile, err
}

func Load(configDir string) (*configfile.ConfigFile, error) {
	if configDir == "" {
		configDir = Dir()
	}
	return load(configDir)
}

func load(configDir string) (*configfile.ConfigFile, error) {
	filename := filepath.Join(configDir, ConfigFileName)
	configFile := configfile.New(filename)

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {

			return configFile, nil
		}

		return configFile, nil
	}
	defer file.Close()
	err = configFile.LoadFromReader(file)
	if err != nil {
		err = errors.Wrap(err, filename)
	}
	return configFile, err
}

// LoadDefaultConfigFile 尝试加载默认配置文件
func LoadDefaultConfigFile(stderr io.Writer) *configfile.ConfigFile {
	configFile, err := load(Dir())
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "WARNING: Error loading config file: %v\n", err)
	}
	return configFile
}
