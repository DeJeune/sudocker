package config

type Mount struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Device      string `json:"device"`
	Fstype      string `json:"fstype"`
	Flags       int    `json:"flags"`
	Data        string `json:"data"`
}
