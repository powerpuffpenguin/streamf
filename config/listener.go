package config

type BasicListener struct {
	// Custom name recorded in logs
	Tag     string `json:"tag"`
	Network string `json:"network"`
	Address string `json:"address"`

	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
	// when one of the two ends of the bridge is disconnected, how long does it take to close the other end? There may still be data in the cache or network at one end of the unbroken connection.
	// In order for this data to be completely forwarded, a reasonable waiting time needs to be set.
	Close string `json:"close"`
}

// Listener to receive incoming traffic
type Listener struct {
	BasicListener
	// work mode, "basic" or "http"
	// default is "basic"
	Mode string `json:"mode"`
	// Specify forwarding destination in "basic" mode
	Dialer Dialer `json:"dialer"`
}
type Dialer struct {
	// connect timeout
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
	// If true, do not verify whether the certificate is valid when connecting to the tls server
	AllowInsecure bool `json:"allowInsecure"`
}
