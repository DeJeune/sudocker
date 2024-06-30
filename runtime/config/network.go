package config

import (
	"encoding/json"
	"net"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

type Network struct {
	// Type sets the networks type, commonly veth and loopback
	Type string `json:"type"`

	// Name of the network interface
	Name string `json:"name"`

	// The bridge to use.
	Bridge string `json:"bridge"`

	// MacAddress contains the MAC address to set on the network interface
	MacAddress string `json:"mac_address"`

	// Address contains the IPv4 and mask to set on the network interface
	Address string `json:"address"`
	SubNet  *net.IPNet
}

// Route defines a routing table entry.
//
// Routes can be specified to create entries in the routing table as the container
// is started.
//
// All of destination, source, and gateway should be either IPv4 or IPv6.
// One of the three options must be present, and omitted entries will use their
// IP family default for the route table.  For IPv4 for example, setting the
// gateway to 1.2.3.4 and the interface to eth0 will set up a standard
// destination of 0.0.0.0(or *) when viewed in the route table.
type Route struct {
	// Destination specifies the destination IP address and mask in the CIDR form.
	Destination string `json:"destination"`

	// Source specifies the source IP address and mask in the CIDR form.
	Source string `json:"source"`

	// Gateway specifies the gateway IP address.
	Gateway string `json:"gateway"`

	// InterfaceName specifies the device to set this route up for, for example eth0.
	InterfaceName string `json:"interface_name"`
}

type Endpoint struct {
	// same with the container's uuid
	Uuid       string           `json:"uuid"`
	IPAddr     net.IP           `json:"ip_addr"`
	Network    *Network         `json:"network"`
	Device     *netlink.Veth    `json:"device"`
	Ports      []string         `json:"ports"`
	MacAddress net.HardwareAddr `json:"mac_addr"`
}

type IPAM struct {
	// the path of ip allocator file.
	Allocator string
	// key is subnet's cidr, value is the bitmap of ipaddr.
	SubnetBitMap *map[string]string
}

type Driver interface {
	Name() string
	Create(subnet string, name string) (*Network, error)
	Delete(nw *Network) error
	Connect(networkName string, ep *Endpoint) error
	Disconnect(epId string) error
}

func (net *Network) Dump(dumpPath string) error {
	if _, err := os.Stat(dumpPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err = os.MkdirAll(dumpPath, 0o644); err != nil {
			return errors.Wrapf(err, "create network dump path %s failed", dumpPath)
		}
	}

	netPath := path.Join(dumpPath, net.Name)
	netFile, err := os.OpenFile(netPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return errors.Wrapf(err, "open file %s failed", dumpPath)
	}
	defer netFile.Close()
	netJson, err := json.Marshal(net)
	if err != nil {
		return errors.Wrapf(err, "Marshal %v failed", net)
	}

	_, err = netFile.Write(netJson)
	return errors.Wrapf(err, "write %s failed", netJson)
}

func (net *Network) Remove(dumpPath string) error {
	// 检查网络对应的配置文件状态，如果文件己经不存在就直接返回
	fullPath := path.Join(dumpPath, net.Name)
	if _, err := os.Stat(fullPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	// 否则删除这个网络对应的配置文件
	return os.Remove(fullPath)
}

func (net *Network) Load(dumpPath string) error {
	// 打开配置文件
	netConfigFile, err := os.Open(dumpPath)
	if err != nil {
		return err
	}
	defer netConfigFile.Close()
	// 从配置文件中读取网络 配置 json 符串
	netJson := make([]byte, 2000)
	n, err := netConfigFile.Read(netJson)
	if err != nil {
		return err
	}

	err = json.Unmarshal(netJson[:n], net)
	return errors.Wrapf(err, "unmarshal %s failed", netJson[:n])
}

type EndpointSettings struct {
	IPAMConfig *EndpointIPAMConfig
	Links      []string
	MacAddress string
}

func (es *EndpointSettings) Copy() *EndpointSettings {
	epCopy := *es
	if es.IPAMConfig != nil {
		epCopy.IPAMConfig = es.IPAMConfig.Copy()
	}

	if es.Links != nil {
		links := make([]string, 0, len(es.Links))
		epCopy.Links = append(links, es.Links...)
	}

	return &epCopy
}

type NetworkingConfig struct {
	Endpoints string // Endpoint configs for each connecting network
}

type EndpointIPAMConfig struct {
	IPv4Address  string   `json:",omitempty"`
	IPv6Address  string   `json:",omitempty"`
	LinkLocalIPs []string `json:",omitempty"`
}

// Copy makes a copy of the endpoint ipam config
func (cfg *EndpointIPAMConfig) Copy() *EndpointIPAMConfig {
	cfgCopy := *cfg
	cfgCopy.LinkLocalIPs = make([]string, 0, len(cfg.LinkLocalIPs))
	cfgCopy.LinkLocalIPs = append(cfgCopy.LinkLocalIPs, cfg.LinkLocalIPs...)
	return &cfgCopy
}
