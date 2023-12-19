package config

import (
	"crypto/tls"
)

type BasicListener struct {
	// Custom name recorded in logs
	Tag     string `json:"tag"`
	Network string `json:"network"`
	Address string `json:"address"`

	TLS TLS `json:"tls"`
}
type TLS struct {
	CertFile string   `json:"certFile"`
	KeyFile  string   `json:"keyFile"`
	Cert     string   `json:"cert"`
	Key      string   `json:"key"`
	Alpn     []string `json:"alpn"`
}

func (t *TLS) Secure() bool {
	return (t.Cert != `` && t.Key != ``) ||
		(t.CertFile != `` && t.KeyFile != ``)
}
func (t *TLS) Certificate() (secure bool, certificate tls.Certificate, alpn []string, e error) {
	if t.Cert != `` && t.Key != `` {
		secure = true
		certificate, e = tls.X509KeyPair([]byte(t.Cert), []byte(t.Key))
	} else if t.CertFile != `` && t.KeyFile != `` {
		secure = true
		certificate, e = tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
	} else {
		return
	}
	if e != nil {
		return
	}
	alpn = t.Alpn
	if len(alpn) == 0 {
		alpn = []string{`h2`, `http/1.1`}
	}
	return
}

// Listener to receive incoming traffic
type Listener struct {
	BasicListener
	// work mode, "basic" or "http"
	// default is "basic"
	Mode string `json:"mode"`
	// Specify forwarding destination in "basic" mode
	Dialer ConnectDialer `json:"dialer"`
	// Specify route for http mode
	Router []*Router `json:"router"`
}
type Router struct {
	// POST PUT PATCH WS
	Method string `json:"method"`
	// url match pattern
	Pattern string `json:"pattern"`
	// Specify forwarding destination
	Dialer ConnectDialer `json:"dialer"`
	// Access token, If non-empty, this value will be verified from the header and url parameters.
	//  * 'ws://example.com/anypath?access_token=access_token=Bearer%20' + rawURLBase64(XXXXX)
	//  * curl -H "Authorization: Bearer " + rawURLBase64(XXXXX)
	Access string `json:"access"`
}
type ConnectDialer struct {
	// Connect dialer with tag 'tcp'
	Tag string `json:"tag"`
	// After one end of the connection is disconnected, wait for one second before closing the other end
	// (waiting for untransmitted data to continue transmitting)
	// Default "1s"
	Close string `json:"close"`
}
