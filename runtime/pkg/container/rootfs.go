package container

import (
	"os"
	"os/exec"
	"strings"

	"github.com/DeJeune/sudocker/runtime/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func NewStorageDriver(containerId, imageName string, binds []string) error {
	if err := createLower(containerId, imageName); err != nil {
		return errors.Errorf("create lower layer failed %v", err)
	}

	if err := createDirs(containerId); err != nil {
		return errors.Errorf("create upper work layer failed %v", err)
	}
	if err := mountOverlayFS(containerId); err != nil {
		return errors.Errorf("mount failed %v", err)
	}
	if len(binds) != 0 {
		for _, bind := range binds {
			mntPath := utils.GetMerged(containerId)
			sourcePath, destinationPath, err := volumeExtract(bind)
			if err != nil {
				return err
			}
			if err := mountVolume(mntPath, sourcePath, destinationPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// volumeExtract 通过冒号分割解析volume目录，比如 -v /tmp:/tmp
func volumeExtract(volume string) (sourcePath, destinationPath string, err error) {
	parts := strings.Split(volume, ":")
	if len(parts) != 2 {
		return "", "", errors.Errorf("invalid volume [%s], must split by `:`", volume)
	}

	sourcePath, destinationPath = parts[0], parts[1]
	if sourcePath == "" || destinationPath == "" {
		return "", "", errors.Errorf("invalid volume [%s], path can't be empty", volume)
	}

	return sourcePath, destinationPath, nil
}

// createLower 将根据containerId和imageName准备
// 目录作为overlayfs的lower层
func createLower(containerId, imageName string) error {
	// 把busybox作为overlayfs中的lower层
	lowerPath := utils.GetLower(containerId)
	imagePath := utils.GetImage(imageName)
	logrus.Infof("lower:%s image.tar:%s", lowerPath, imagePath)
	// 检查是否已经存在lower目录
	exist, err := utils.PathExists(lowerPath)
	if err != nil {
		logrus.Infof("Fail to judge whether dir %s exists. %v", lowerPath, err)
	}
	// 不存在则创建目录并将busybox.tar解压到busybox文件夹中
	if !exist {
		if err := os.MkdirAll(lowerPath, 0777); err != nil {
			return errors.Errorf("Mkdir dir %s error. %v", lowerPath, err)
		}
		if _, err := exec.Command("tar", "-xvf", imagePath, "-C", lowerPath).CombinedOutput(); err != nil {
			return errors.Errorf("Untar dir %s error %v", lowerPath, err)
		}
	}
	return nil
}

// createDirs 创建overlayfs需要的的merged、upper、worker目录
func createDirs(containerId string) error {
	dirs := []string{
		utils.GetMerged(containerId),
		utils.GetUpper(containerId),
		utils.GetWorker(containerId),
	}
	for _, dir := range dirs {
		if err := os.Mkdir(dir, 0o777); err != nil {
			return errors.Errorf("mkdir dir %s error. %v", dir, err)
		}
	}
	return nil
}

// mountOverlayFS 挂载overlayfs
func mountOverlayFS(containerId string) error {
	// mount -t overlay overlay -o lowerdir=lower1:lower2:lower3,upperdir=upper,workdir=work merged
	dirs := utils.GetOverlayFSDirs(utils.GetLower(containerId), utils.GetUpper(containerId), utils.GetWorker(containerId))
	mergedPath := utils.GetMerged(containerId)
	if err := mount("overlay", mergedPath, "overlay", 0, dirs); err != nil {
		return errors.Errorf("%v", err)
	}
	return nil
}

// DeleteStorageDriver Delete the AUFS filesystem while container exit
func DeleteStorageDriver(containerId string, volumes []string) error {
	if len(volumes) != 0 {
		for _, volume := range volumes {
			_, containerPath, err := volumeExtract(volume)
			if err != nil {
				return errors.Errorf("extract volume failed, maybe volume parameter input is not correct, detail: %v", err)
			}
			mntPath := utils.GetMerged(containerId)
			if err := umountVolume(mntPath, containerPath); err != nil {
				return errors.Errorf("umount volume failed: %v", err)
			}
		}
	}
	if err := umountOverlayFS(containerId); err != nil {
		return errors.Errorf("umount overlay2 fs failed: %v", err)
	}
	if err := deleteDirs(containerId); err != nil {
		return errors.Errorf("delete dirs failed: %v", err)
	}
	return nil
}

func umountOverlayFS(containerId string) error {
	mntPath := utils.GetMerged(containerId)
	if err := unmount(mntPath, 0); err != nil {
		return errors.Errorf("umount failed %v", err)
	}
	if err := os.RemoveAll(mntPath); err != nil {
		return errors.Errorf("Remove dir %s error %v", mntPath, err)
	}
	return nil
}

func deleteDirs(containerId string) error {
	dirs := []string{
		utils.GetMerged(containerId),
		utils.GetUpper(containerId),
		utils.GetWorker(containerId),
		utils.GetLower(containerId),
		utils.GetRoot(containerId), // root 目录也要删除
	}
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			logrus.Errorf("Remove dir %s error %v", dir, err)
		}
	}
	return nil
}
