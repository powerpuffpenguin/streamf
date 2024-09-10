package config

type UDP struct {
	// udp timeout, default 60s
	Timeout string `json:"timeout"`
	// udp max frame length, default 1024*2
	Size int `json:"size"`
	// frame buffer, default 16
	Frame int `json:"frame"`
}
