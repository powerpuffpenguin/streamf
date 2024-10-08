package config

type UDPForward struct {
	// log tag
	Tag string `json:"tag"`
	// "udp" "udp4" "udp6"
	Network string `json:"network"`
	// udp listen host:port
	Listen string `json:"listen"`
	// remote target addr
	To string `json:"to"`
	// udp max frame length, default 1024*2
	Size int `json:"size"`
	// udp timeout, default 3m
	Timeout string `json:"timeout"`
}
