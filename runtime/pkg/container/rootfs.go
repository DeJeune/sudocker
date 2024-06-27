package container

import (
	"os"
	"os/exec"
	"strings"

	"github.com/DeJeune/sudocker/runtime/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// /root /root/merged/
func NewStorageDriver(rootPath, mntPath string, binds []string) error {
	if err := createLower(rootPath); err != nil {
		return errors.Errorf("create lower layer failed %v", err)
	}

	if err := createDirs(rootPath); err != nil {
		return errors.Errorf("create upper work layer failed %v", err)
	}
	if err := mountOverlayFS(rootPath, mntPath); err != nil {
		return errors.Errorf("mount failed %v", err)
	}
	if len(binds) != 0 {
		for _, bind := range binds {
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

// createLower 将busybox作为overlayfs的lower层
func createLower(rootURL string) error {
	// 把busybox作为overlayfs中的lower层
	busyboxURL := rootURL + "busybox/"
	busyboxTarURL := rootURL + "busybox.tar"
	// 检查是否已经存在busybox文件夹
	exist, err := utils.PathExists(busyboxURL)
	if err != nil {
		logrus.Infof("Fail to judge whether dir %s exists. %v", busyboxURL, err)
	}
	// 不存在则创建目录并将busybox.tar解压到busybox文件夹中
	if !exist {
		if err := os.Mkdir(busyboxURL, 0777); err != nil {
			return errors.Errorf("Mkdir dir %s error. %v", busyboxURL, err)
		}
		if _, err := exec.Command("tar", "-xvf", busyboxTarURL, "-C", busyboxURL).CombinedOutput(); err != nil {
			return errors.Errorf("Untar dir %s error %v", busyboxURL, err)
		}
	}
	return nil
}

// createDirs 创建overlayfs需要的的upper、worker目录
func createDirs(rootURL string) error {
	upperURL := rootURL + "upper/"
	if err := os.Mkdir(upperURL, 0o777); err != nil {
		return errors.Errorf("mkdir dir %s error. %v", upperURL, err)
	}
	workURL := rootURL + "work/"
	if err := os.Mkdir(workURL, 0o777); err != nil {
		return errors.Errorf("mkdir dir %s error. %v", workURL, err)
	}
	return nil
}

// mountOverlayFS 挂载overlayfs
func mountOverlayFS(rootURL, mntURL string) error {
	// mount -t overlay overlay -o lowerdir=lower1:lower2:lower3,upperdir=upper,workdir=work merged
	// 创建对应的挂载目录
	if err := os.Mkdir(mntURL, 0o777); err != nil {
		return errors.Errorf("Mkdir dir %s error. %v", mntURL, err)
	}
	// 拼接参数
	// e.g. lowerdir=/root/busybox,upperdir=/root/upper,workdir=/root/merged
	dirs := "lowerdir=" + rootURL + "busybox" + ",upperdir=" + rootURL + "upper" + ",workdir=" + rootURL + "work"
	// dirs := "dirs=" + rootURL + "writeLayer:" + rootURL + "busybox"
	if err := mount("overlay", mntURL, "overlay", 0, dirs); err != nil {
		return errors.Errorf("%v", err)
	}
	return nil
}

// DeleteStorageDriver Delete the AUFS filesystem while container exit
func DeleteStorageDriver(rootURL string, mntURL string) {
	umountOverlayFS(mntURL)
	deleteDirs(rootURL)
}

func umountOverlayFS(mntURL string) error {
	if err := unmount(mntURL, 0); err != nil {
		return errors.Errorf("unmount failed %v", err)
	}
	if err := os.RemoveAll(mntURL); err != nil {
		return errors.Errorf("Remove dir %s error %v", mntURL, err)
	}
	return nil
}

func deleteDirs(rootURL string) error {
	writeURL := rootURL + "upper/"
	if err := os.RemoveAll(writeURL); err != nil {
		return errors.Errorf("Remove dir %s error %v", writeURL, err)
	}
	workURL := rootURL + "work"
	if err := os.RemoveAll(workURL); err != nil {
		return errors.Errorf("Remove dir %s error %v", workURL, err)
	}
	return nil
}
