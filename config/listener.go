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
	// Specify route for http mode
	Router []*Router `json:"router"`
}
type Router struct {
	// POST PUT PATCH WS
	Method string `json:"method"`
	// url match pattern
	Pattern string `json:"pattern"`
	// Specify forwarding destination
	Dialer string `json:"dialer"`
	// Access token, If non-empty, this value will be verified from the header and url parameters.
	//  * 'ws://example.com/anypath?access_token=access_token=Bearer%20' + rawURLBase64(XXXXX)
	//  * curl -H "Authorization: Bearer " + rawURLBase64(XXXXX)
	Access string `json:"access"`
}
