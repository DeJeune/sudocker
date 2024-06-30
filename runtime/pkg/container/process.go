package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/DeJeune/sudocker/runtime/utils"
	"github.com/pkg/errors"
)

func RecordContainerInfo(containerInfo *Info) error {
	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		return errors.WithMessage(err, "container info marshal failed")
	}
	jsonStr := string(jsonBytes)
	// 拼接出存储容器信息文件的路径，如果目录不存在则级联创建
	dirPath := fmt.Sprintf(utils.InfoLocFormat, containerInfo.Id)
	if err := os.MkdirAll(dirPath, 0o777); err != nil {
		return errors.WithMessagef(err, "mkdir %s failed", dirPath)
	}
	// 将容器信息写入文件
	fileName := path.Join(dirPath, utils.ConfigName)
	file, err := os.Create(fileName)
	if err != nil {
		return errors.Errorf("create file failed: %v", err)
	}
	defer file.Close()
	// if err != nil {
	// 	return errors.WithMessagef(err, "create file %s failed", fileName)
	// }
	if _, err = file.WriteString(jsonStr); err != nil {
		return errors.WithMessagef(err, "write container info to  file %s failed", fileName)
	}
	return nil
}

func GenerateContainerID() string {
	return utils.RandStringBytes(utils.IDLength)
}

func DeleteContainerInfo(containerID string) error {
	dirPath := fmt.Sprintf(utils.InfoLocFormat, containerID)
	if err := os.RemoveAll(dirPath); err != nil {
		return errors.Errorf("Remove dir %s error: %v", dirPath, err)
	}
	return nil
}
