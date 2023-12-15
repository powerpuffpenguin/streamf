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
	// Default "1s"
	Close string `json:"close"`
}

// Listener to receive incoming traffic
type Listener struct {
	BasicListener
	// work mode, "basic" or "http"
	// default is "basic"
	Mode string `json:"mode"`
	// Specify forwarding destination in "basic" mode
	Dialer string `json:"dialer"`
}
