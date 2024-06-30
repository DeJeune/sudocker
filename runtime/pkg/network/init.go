package network

import (
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/DeJeune/sudocker/runtime/pkg/container"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

var (
	defaultNetworkPath = "/var/lib/sudocker/network/network/"
	drivers            = map[string]config.Driver{}
)

func init() {
	var bridgeDriver = BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = &bridgeDriver
	if _, err := os.Stat(defaultNetworkPath); err != nil {
		if !os.IsNotExist(err) {
			logrus.Errorf("check %s is exist failed, detail:%v", defaultNetworkPath, err)
		}
		if err = os.MkdirAll(defaultNetworkPath, 0o644); err != nil {
			logrus.Errorf("create %s failed,detail:%v", defaultNetworkPath, err)
			return
		}
	}

}

func loadNetwork() (map[string]*config.Network, error) {
	networks := map[string]*config.Network{}
	err := filepath.Walk(defaultNetworkPath, func(netPath string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		_, netName := path.Split(netPath)
		net := &config.Network{
			Name: netName,
		}
		if err = net.Load(netPath); err != nil {
			logrus.Errorf("error load network: %s", err)
		}
		networks[netName] = net
		return nil
	})
	return networks, err
}

// CreateNetwork 根据不同 driver 创建 Network
func CreateNetwork(driver, subnet, name string) error {
	// 将网段的字符串转换成net. IPNet的对象
	_, cidr, _ := net.ParseCIDR(subnet)
	// 通过IPAM分配网关IP，获取到网段中第一个IP作为网关的IP
	ip, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = ip
	// 调用指定的网络驱动创建网络，这里的 drivers 字典是各个网络驱动的实例字典 通过调用网络驱动
	// Create 方法创建网络，后面会以 Bridge 驱动为例介绍它的实现
	net, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}
	// 保存网络信息，将网络的信息保存在文件系统中，以便查询和在网络上连接网络端点
	return net.Dump(defaultNetworkPath)
}

func ContainsNetwork(name string) (bool, error) {
	networks, err := loadNetwork()
	if err != nil {
		return false, errors.Errorf("load network from file failed,detail: %v", err)
	}
	for _, network := range networks {
		if network.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// ListNetwork 打印出当前全部 Network 信息
func ListNetwork() error {
	networks, err := loadNetwork()
	if err != nil {
		return errors.Errorf("load network from file failed,detail: %v", err)
	}
	// 通过tabwriter库把信息打印到屏幕上
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tIpRange\tDriver\n")
	for _, net := range networks {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			net.Name,
			net.Address,
			net.Bridge,
		)
	}
	if err = w.Flush(); err != nil {
		return errors.Errorf("Flush error %v", err)
	}
	return nil
}

// DeleteNetwork 根据名字删除 Network
func DeleteNetwork(networkName string) error {
	networks, err := loadNetwork()
	if err != nil {
		return errors.WithMessage(err, "load network from file failed")
	}
	// 网络不存在直接返回一个error
	net, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("no Such Network: %s", networkName)
	}
	// 调用IPAM的实例ipAllocator释放网络网关的IP
	if err = ipAllocator.Release(net.SubNet, &net.SubNet.IP); err != nil {
		return errors.Wrap(err, "remove Network gateway ip failed")
	}
	// 调用网络驱动删除网络创建的设备与配置
	if err = drivers[net.Bridge].Delete(net); err != nil {
		return errors.Wrap(err, "remove Network DriverError failed")
	}
	// 最后从网络的目录中删除该网络对应的配置文件
	return net.Remove(defaultNetworkPath)
}

// Connect 连接容器到之前创建的网络 mydocker run -net testnet -p 8080:80 xxxx
func Connect(networkName string, info *container.Info) (net.IP, error) {
	networks, err := loadNetwork()
	if err != nil {
		return nil, errors.WithMessage(err, "load network from file failed")
	}
	// 从networks字典中取到容器连接的网络的信息，networks字典中保存了当前己经创建的网络
	network, ok := networks[networkName]
	logrus.Infof("network structrue: %v", network)
	if !ok {
		return nil, fmt.Errorf("no Such Network: %s", networkName)
	}

	// 分配容器IP地址
	ip, err := ipAllocator.Allocate(network.SubNet)
	if err != nil {
		return ip, errors.Wrapf(err, "allocate ip")
	}
	// 创建网络端点
	ep := &config.Endpoint{
		Uuid:    fmt.Sprintf("%s-%s", info.Id, networkName),
		IPAddr:  ip,
		Network: network,
		Ports:   info.PortMapping,
	}
	// 调用网络驱动挂载和配置网络端点
	if err = drivers[network.Bridge].Connect(network.Name, ep); err != nil {
		return ip, err
	}
	// 到容器的namespace配置容器网络设备IP地址
	if err = configEndpointIpAddressAndRoute(ep, info); err != nil {
		return ip, err
	}
	// 配置端口映射信息，例如 mydocker run -p 8080:80
	return ip, addPortMapping(ep)
}

func addPortMapping(ep *config.Endpoint) error {
	return configPortMapping(ep, false)
}

// configPortMapping 配置端口映射
func configPortMapping(ep *config.Endpoint, isDelete bool) error {
	action := "-A"
	if isDelete {
		action = "-D"
	}

	var err error
	// 遍历容器端口映射列表
	for _, pm := range ep.Ports {
		// 分割成宿主机的端口和容器的端口
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			logrus.Errorf("port mapping format error, %v", pm)
			continue
		}
		// 由于iptables没有Go语言版本的实现，所以采用exec.Command的方式直接调用命令配置
		// 在iptables的PREROUTING中添加DNAT规则
		// 将宿主机的端口请求转发到容器的地址和端口上
		// iptables -t nat -A PREROUTING ! -i testbridge -p tcp -m tcp --dport 8080 -j DNAT --to-destination 10.0.0.4:80
		iptablesCmd := fmt.Sprintf("-t nat %s PREROUTING ! -i %s -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s",
			action, ep.Network.Name, portMapping[0], ep.IPAddr.String(), portMapping[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		logrus.Infoln("配置端口映射 DNAT cmd:", cmd.String())
		// 执行iptables命令,添加端口映射转发规则
		output, err := cmd.Output()
		if err != nil {
			logrus.Errorf("iptables Output, %v", output)
			continue
		}
	}
	return err
}

func enterContainerNetNS(enLink *netlink.Link, info *container.Info) func() {
	// 找到容器的Net Namespace
	// /proc/[pid]/ns/net 打开这个文件的文件描述符就可以来操作Net Namespace
	// 而ContainerInfo中的PID,即容器在宿主机上映射的进程ID
	// 它对应的/proc/[pid]/ns/net就是容器内部的Net Namespace
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", info.Pid), os.O_RDONLY, 0)
	if err != nil {
		logrus.Errorf("error get container net namespace, %v", err)
	}

	nsFD := f.Fd()
	// 锁定当前程序所执行的线程，如果不锁定操作系统线程的话
	// Go语言的goroutine可能会被调度到别的线程上去
	// 就不能保证一直在所需要的网络空间中了
	// 所以先调用runtime.LockOSThread()锁定当前程序执行的线程
	runtime.LockOSThread()

	// 修改网络端点Veth的另外一端，将其移动到容器的Net Namespace 中
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		logrus.Errorf("error set link netns , %v", err)
	}

	// 获取当前的网络namespace
	origns, err := netns.Get()
	if err != nil {
		logrus.Errorf("error get current netns, %v", err)
	}

	// 调用 netns.Set方法，将当前进程加入容器的Net Namespace
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		logrus.Errorf("error set netns, %v", err)
	}
	// 返回之前Net Namespace的函数
	// 在容器的网络空间中执行完容器配置之后调用此函数就可以将程序恢复到原生的Net Namespace
	return func() {
		// 恢复到上面获取到的之前的 Net Namespace
		netns.Set(origns)
		origns.Close()
		// 取消对当附程序的线程锁定
		runtime.UnlockOSThread()
		f.Close()
	}
}

// configEndpointIpAddressAndRoute 配置容器网络端点的地址和路由
func configEndpointIpAddressAndRoute(ep *config.Endpoint, info *container.Info) error {
	// 根据名字找到对应Veth设备
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return errors.WithMessagef(err, "found veth [%s] failed", ep.Device.PeerName)
	}
	// 将容器的网络端点加入到容器的网络空间中
	// 并使这个函数下面的操作都在这个网络空间中进行
	// 执行完函数后，恢复为默认的网络空间，具体实现下面再做介绍

	defer enterContainerNetNS(&peerLink, info)()
	// 获取到容器的IP地址及网段，用于配置容器内部接口地址
	// 比如容器IP是192.168.1.2， 而网络的网段是192.168.1.0/24
	// 那么这里产出的IP字符串就是192.168.1.2/24，用于容器内Veth端点配置

	interfaceIP := *ep.Network.SubNet
	interfaceIP.IP = ep.IPAddr
	// 设置容器内Veth端点的IP
	if err = setInterfaceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("%v,%s", ep.Network, err)
	}
	// 启动容器内的Veth端点
	if err = setInterfaceUP(ep.Device.PeerName); err != nil {
		return err
	}
	// Net Namespace 中默认本地地址 127 的勺。”网卡是关闭状态的
	// 启动它以保证容器访问自己的请求
	if err = setInterfaceUP("lo"); err != nil {
		return err
	}
	// 设置容器内的外部请求都通过容器内的Veth端点访问
	// 0.0.0.0/0的网段，表示所有的IP地址段
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	// 构建要添加的路由数据，包括网络设备、网关IP及目的网段
	// 相当于route add -net 0.0.0.0/0 gw (Bridge网桥地址) dev （容器内的Veth端点设备)

	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        ep.Network.SubNet.IP,
		Dst:       cidr,
	}
	// 调用netlink的RouteAdd,添加路由到容器的网络空间
	// RouteAdd 函数相当于route add 命令
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}

	return nil
}
