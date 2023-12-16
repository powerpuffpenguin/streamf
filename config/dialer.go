package config

type Dialer struct {
	// Must be unique
	Tag string `json:"tag"`
	// Connect timeout
	// Default 500ms
	Timeout string `json:"timeout"`
	// connect url
	//  * "ws://host/path"
	//  * "wss://host/path"
	//  * "http://host/path"
	//  * "https://host/path"
	//  * "tcp://host:port"
	//  * "tcp+tls://host:port"
	//  * "unix://host"
	//  * "unix+tls://host"
	URL string `json:"url"`
	// optional connect address
	Addr string `json:"addr"`
	// optional network
	Network string `json:"network"`
	// If true, do not verify whether the certificate is valid when connecting to the tls server
	AllowInsecure bool `json:"allowInsecure"`

	// http method, default "PUT"
	Method string
}
