package config

type Logger struct {
	Level string `json:"level"`
	// add source
	Source bool `json:"source"`
}
