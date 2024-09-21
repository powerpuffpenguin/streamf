package config

import (
	"encoding/json"
	"fmt"

	"github.com/google/go-jsonnet"
)

type Config struct {
	Logger Logger `json:"logger"`
	Pool   Pool   `json:"pool"`
	// Listener to receive incoming traffic
	Listener []*Listener `json:"listener"`
	// Outgoing traffic, how to connect to the remote end
	Dialer []*Dialer `json:"dialer"`
	// A reverse bridge
	Bridge []*Bridge `json:"bridge"`
	// udp forward
	UDP []*UDPForward `json:"udp"`
}

func (c *Config) Load(filename string) (e error) {
	vm := jsonnet.MakeVM()
	jsonStr, e := vm.EvaluateFile(filename)
	if e != nil {
		return
	}
	e = json.Unmarshal([]byte(jsonStr), c)
	if e != nil {
		return
	}
	return
}
func (c *Config) Print(filename string) (e error) {
	vm := jsonnet.MakeVM()
	jsonStr, e := vm.EvaluateFile(filename)
	if e != nil {
		return
	}
	fmt.Print(jsonStr)
	return
}
