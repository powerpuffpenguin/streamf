package config

type Socks struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Connect  string `json:"connect"`
}
