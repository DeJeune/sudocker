package container

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type RemoveOptions struct {
	RemoveVolumes bool
	RemoveLinks   bool
	Force         bool
}

func RmContainer(containerId string, opts RemoveOptions) error {
	force := opts.Force
	containerId = strings.Trim(containerId, "/")
	if containerId == "" {
		return errors.New("Container name cannot be empty")
	}

	containerInfo, err := GetInfoByContainerId(containerId)
	if err != nil {
		return errors.Errorf("Get container %s info error %v", containerId, err)
	}

	switch containerInfo.Status {
	case Stopped: // STOP 状态容器直接删除即可
		// 先删除配置目录，再删除rootfs 目录
		if err = DeleteContainerInfo(containerId); err != nil {
			logrus.Errorf("Remove container [%s]'s config failed, detail: %v", containerId, err)
		}
		DeleteStorageDriver(containerId, containerInfo.Volumes)
		// if containerInfo.NetworkName != "" { // 清理网络资源
		// 	if err = network.Disconnect(containerInfo.NetworkName, containerInfo); err != nil {
		// 		log.Errorf("Remove container [%s]'s config failed, detail: %v", containerId, err)
		// 		return
		// 	}
		// }
	case Running: // RUNNING 状态容器如果指定了 force 则先 stop 然后再删除
		if !force {
			return errors.Errorf("Couldn't remove running container [%s], Stop the container before attempting removal or"+
				" force remove", containerId)
		}
		logrus.Infof("force delete running container [%s]", containerId)
		if err := StopContainer(containerId); err != nil {
			return errors.Errorf("stop a running container failed: %v", err)
		}
		RmContainer(containerId, opts)
	default:
		return errors.Errorf("Couldn't remove container,invalid status %s", containerInfo.Status)
	}
	return nil
}
