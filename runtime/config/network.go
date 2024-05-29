package config

import "net"

type Network struct {
	Name       string     `json:"Name"`
	Counts     uint32     `json:"Counts"`
	Driver     string     `json:"Driver"`
	CreateTime string     `json:"CreateTime"`
	IPNet      *net.IPNet `json:"IPNet"`
	Gateway    *net.IPNet `json:"Gateway"`
}
