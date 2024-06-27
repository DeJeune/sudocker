package container

import (
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// mountVolume 使用 bind mount 挂载 volume
func mountVolume(mntPath, hostPath, containerPath string) error {
	// 创建宿主机目录
	if err := os.MkdirAll(hostPath, 0o777); err != nil {
		return errors.Errorf("mkdir parent dir %s error. %v", hostPath, err)
	}
	// 拼接出对应的容器目录在宿主机上的的位置，并创建对应目录
	containerPathInHost := path.Join(mntPath, containerPath)
	logrus.Infof("containerPathInHost: %s", containerPathInHost)
	if err := os.MkdirAll(containerPathInHost, 0o777); err != nil {
		return errors.Errorf("mkdir container dir %s error. %v", containerPathInHost, err)
	}
	// 通过bind mount 将宿主机目录挂载到容器目录
	// mount -o bind /hostPath /containerPath

	if err := mount(hostPath, containerPathInHost, "bind", unix.MS_BIND, ""); err != nil {
		return errors.Errorf("mount volume failed. %v", err)
	}
	return nil
}

func DeleteVolumes(rootPath string, volumes []string) error {
	mntPath := path.Join(rootPath, "merged")

	// 如果指定了volume则需要umount volume
	// NOTE: 一定要要先 umount volume ，然后再删除目录，否则由于 bind mount 存在，删除临时目录会导致 volume 目录中的数据丢失。
	if len(volumes) != 0 {
		for _, volume := range volumes {
			_, containerPath, err := volumeExtract(volume)
			if err != nil {
				return errors.Errorf("extract volume failed，maybe volume parameter input is not correct，detail:%v", err)

			}
			if err := unmountVolume(mntPath, containerPath); err != nil {
				return err
			}
		}
	}

	if err := umountOverlayFS(mntPath); err != nil {
		return err
	}
	if err := deleteDirs(rootPath); err != nil {
		return err
	}
	return nil
}

func unmountVolume(mntPath, containerPath string) error {
	// mntPath 为容器在宿主机上的挂载点，例如 /root/merged
	// containerPath 为 volume 在容器中对应的目录，例如 /root/tmp
	// containerPathInHost 则是容器中目录在宿主机上的具体位置，例如 /root/merged/root/tmp
	containerPathInHost := path.Join(mntPath, containerPath)
	if err := unmount(containerPathInHost, 0); err != nil {
		return errors.Errorf("Umount volume failed. %v", err)
	}
	return nil
}
