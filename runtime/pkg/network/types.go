package network

import (
	"net"

	"github.com/DeJeune/sudocker/runtime/config"
	"github.com/vishvananda/netlink"
)

type Endpoint struct {
	// same with the container's uuid
	Uuid    string            `json:"Uuid"`
	IPAddr  net.IP            `json:"IPAddr"`
	Network *config.Network   `json:"Network"`
	Device  *netlink.Veth     `json:"Device"`
	Ports   map[string]string `json:"Ports"`
}

type IPAM struct {
	// the path of ip allocator file.
	Allocator string
	// key is subnet's cidr, value is the bitmap of ipaddr.
	SubnetBitMap *map[string]string
}

type Driver interface {
	Name() string
	Init(nw *config.Network) error
	Create(nw *config.Network) error
	Delete(nw *config.Network) error
	Connect(ep *Endpoint) error
	DisConnect(ep *Endpoint) error
}
