package container

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/sirupsen/logrus"
)

func setupMount() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}
	logrus.Infof("Current location is %s", pwd)
	if err = mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return err
	}
	if err = pivotRoot(pwd); err != nil {
		return err
	}
	// 挂载/proc文件系统
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	if err = mount("proc", "/proc", "proc", uintptr(defaultMountFlags), ""); err != nil {
		return err
	}
	// 由于前面 pivotRoot 切换了 rootfs，因此这里重新 mount 一下 /dev 目录
	// tmpfs 是基于 件系 使用 RAM、swap 分区来存储。
	// 不挂载 /dev，会导致容器内部无法访问和使用许多设备，这可能导致系统无法正常工作
	if err = mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755"); err != nil {
		return err
	}
	return nil
}

func pivotRoot(root string) error {
	/**
	  NOTE：PivotRoot调用有限制，newRoot和oldRoot不能在同一个文件系统下。
	  因此，为了使当前root的老root和新root不在同一个文件系统下，这里把root重新mount了一次。
	  bind mount是把相同的内容换了一个挂载点的挂载方法
	*/
	if err := mount(root, root, "bind", unix.MS_BIND|unix.MS_REC, ""); err != nil {
		return errors.Wrap(err, "mount rootfs to itself")
	}
	// 创建 rootfs/.pivot_root 目录用于存储 old_root
	pivotDir := filepath.Join(root, ".pivot_root")
	if err := os.Mkdir(pivotDir, 0777); err != nil {
		return err
	}
	// 执行pivot_root调用,将系统rootfs切换到新的rootfs,
	// PivotRoot调用会把 old_root挂载到pivotDir,也就是rootfs/.pivot_root,挂载点现在依然可以在mount命令中看到
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		return errors.WithMessagef(err, "pivotRoot failed,new_root:%v old_put:%v", root, pivotDir)
	}
	// 修改当前的工作目录到根目录
	if err := syscall.Chdir("/"); err != nil {
		return errors.WithMessage(err, "chdir to / failed")
	}

	// 最后再把old_root umount了，即 umount rootfs/.pivot_root
	// 由于当前已经是在 rootfs 下了，就不能再用上面的rootfs/.pivot_root这个路径了,现在直接用/.pivot_root这个路径即可
	pivotDir = filepath.Join("/", ".pivot_root")
	if err := unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return errors.WithMessage(err, "unmount pivot_root dir")
	}
	// 删除临时文件夹
	return os.Remove(pivotDir)
}

func RunContainerInitProcess() error {
	// mount /proc文件系统
	if err := setupMount(); err != nil {
		return errors.Errorf("mount failed. %v", err)
	}
	cmdArray := readUserCommand()
	if len(cmdArray) == 0 {
		return errors.New("run container get user command error, cmdArray is nil")
	}
	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		logrus.Errorf("Exec loop path error %v", err)
		return err
	}
	logrus.Infof("Find path %s", path)
	if err := syscall.Exec(path, cmdArray[0:], os.Environ()); err != nil {
		logrus.Errorf("RunContainerInitProcess exec :" + err.Error())
	}

	return nil
}

func readUserCommand() []string {
	// uintptr(3 ）就是指 index 为3的文件描述符，也就是传递进来的管道的另一端，至于为什么是3，具体解释如下：
	/*	因为每个进程默认都会有3个文件描述符，分别是标准输入、标准输出、标准错误。这3个是子进程一创建的时候就会默认带着的，
		前面通过ExtraFiles方式带过来的 readPipe 理所当然地就成为了第4个。
		在进程中可以通过index方式读取对应的文件，比如
		index0：标准输入
		index1：标准输出
		index2：标准错误
		index3：带过来的第一个FD，也就是readPipe
		由于可以带多个FD过来，所以这里的3就不是固定的了。
		比如像这样：cmd.ExtraFiles = []*os.File{a,b,c,readPipe} 这里带了4个文件过来，分别的index就是3,4,5,6
		那么我们的 readPipe 就是 index6,读取时就要像这样：pipe := os.NewFile(uintptr(6), "pipe")
	*/
	pipe := os.NewFile(uintptr(3), "pipe")
	msg, err := io.ReadAll(pipe)
	if err != nil {
		logrus.Errorf("init read pipe error %v", err)
		return nil
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")
}
