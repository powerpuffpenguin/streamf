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
	URL string `json:"url"`
	// optional connect address
	Addr string `json:"addr"`
	// optional network
	Network string `json:"network"`
	// If true, do not verify whether the certificate is valid when connecting to the tls server
	AllowInsecure bool `json:"allowInsecure"`
	// If dialing fails, how many times to retry
	Retry int `json:"retry"`
	// http method, default "POST"
	Method string `json:"method"`
}
