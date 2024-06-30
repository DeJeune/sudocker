package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"syscall"

	"github.com/DeJeune/sudocker/runtime/utils"
	"github.com/pkg/errors"
)

func StopContainer(containerId string) error {
	containerInfo, err := GetInfoByContainerId(containerId)
	if err != nil {
		return errors.Errorf("Get container %s info error %v", containerId, err)
	}
	pidInt, err := strconv.Atoi(containerInfo.Pid)
	if err != nil {
		return errors.Errorf("Conver pid from string to int error %v", err)
	}
	if err := syscall.Kill(pidInt, syscall.SIGTERM); err != nil {
		return errors.Errorf("Stop container %s error %v", containerId, err)
	}
	// 修改容器信息
	containerInfo.Status = Stopped
	containerInfo.Pid = ""
	newContentBytes, err := json.Marshal(containerInfo)
	if err != nil {
		return errors.Errorf("Json marshal %s error %v", containerId, err)
	}
	dirPath := fmt.Sprintf(utils.InfoLocFormat, containerId)
	configFilePath := path.Join(dirPath, utils.ConfigName)
	if err = os.WriteFile(configFilePath, newContentBytes, 0o622); err != nil {
		return errors.Errorf("Write file %s error:%v", configFilePath, err)
	}
	return nil
}

func GetInfoByContainerId(containerId string) (*Info, error) {
	dirPath := fmt.Sprintf(utils.InfoLocFormat, containerId)
	configFilePath := path.Join(dirPath, utils.ConfigName)
	contentBytes, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, errors.Wrapf(err, "read file %s", configFilePath)
	}
	var containerInfo Info
	if err = json.Unmarshal(contentBytes, &containerInfo); err != nil {
		return nil, err
	}
	return &containerInfo, nil
}
