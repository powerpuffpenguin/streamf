package config

import (
	"encoding/json"

	"github.com/google/go-jsonnet"
)

type Config struct {
	Logger Logger `json:"logger"`
	// Listener to receive incoming traffic
	Listener []*Listener `json:"listener"`
	// Outgoing traffic, how to connect to the remote end
	Dialer []*Dialer `json:"dialer"`
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
