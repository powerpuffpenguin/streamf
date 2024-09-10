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
	//  * "basic://host:port"
	//  * "basic+tls://host:port"
	URL string `json:"url"`
	// optional connect address
	Addr string `json:"addr"`
	// optional network
	Network string `json:"network"`
	// If true, only websocket handshake is used, and tcp communication is used directly after the handshake is successful.
	Fast bool `json:"fast"`
	// If true, do not verify whether the certificate is valid when connecting to the tls server
	AllowInsecure bool `json:"allowInsecure"`
	// If dialing fails, how many times to retry
	Retry int `json:"retry"`
	// http method, default "POST"
	Method string `json:"method"`
	// Optional credentials, only valid for http protocol
	Access string `jsong:"access"`

	// ping http default '40s'
	Ping string `json:"ping"`

	// http client header
	Header map[string][]string `json:"header"`
}
type Bridge struct {
	Tag string `json:"tag"`
	// connect url
	//  * "ws://host/path"
	//  * "wss://host/path"
	//  * "http://host/path"
	//  * "https://host/path"
	//  * "basic://host:port"
	//  * "basic+tls://host:port"
	URL string `json:"url"`
	// optional connect address
	Addr string `json:"addr"`
	// optional network
	Network string `json:"network"`
	// If true, only websocket handshake is used, and tcp communication is used directly after the handshake is successful.
	Fast bool `json:"fast"`
	// If true, do not verify whether the certificate is valid when connecting to the tls server
	AllowInsecure bool `json:"allowInsecure"`
	// http method, default "POST"
	Method string `json:"method"`
	// Optional credentials, only valid for http protocol
	Access string `jsong:"access"`
	// Specify forwarding destination in "basic" mode
	Dialer ConnectDialer `json:"dialer"`

	// ping http default '40s'
	Ping string `json:"ping"`
}
