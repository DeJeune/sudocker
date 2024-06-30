package container

import (
	"fmt"
	"path"
	"testing"

	"github.com/DeJeune/sudocker/runtime/utils"
)

func TestList(t *testing.T) {
	fileName := "12345678"
	// 根据文件名拼接出完整路径
	configFileDir := fmt.Sprintf(utils.InfoLocFormat, fileName)
	configFileDir = path.Join(configFileDir, utils.ConfigName)
	expected := "/var/lib/sudocker/containers/12345678/config.json"
	if configFileDir != expected {
		t.Errorf("expected %s but %s got", expected, configFileDir)
	}
}
