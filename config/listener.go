package config

import (
	"crypto/tls"
)

type BasicListener struct {
	// Custom name recorded in logs
	Tag     string `json:"tag"`
	Network string `json:"network"`
	Addr    string `json:"addr"`

	TLS TLS `json:"tls"`
	// udp settings
	UDP UDP `json:"udp"`
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
	// work mode, "basic" or "http" or "portal"
	// default is "basic"
	Mode string `json:"mode"`
	// Specify forwarding destination in "basic" mode
	Dialer ConnectDialer `json:"dialer"`
	// Specify route for http mode
	Router []*Router `json:"router"`
	Portal Portal    `json:"portal"`
	// udp settings
	UDP UDP `json:"udp"`
}
type Portal struct {
	Tag string `json:"tag"`
	// Wait connect timeout
	// Default 500ms
	Timeout string `json:"timeout"`
	// How often does an idle connection send a heartbeat?
	Heart string `json:"heart"`
	// Timeout for waiting for heartbeat response
	HeartTimeout string `json:"heartTimeout"`
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

	// basic auth
	Auth []BasicAuth `json:"auth"`

	// Portal tag, if not emoty, enable portal mode
	Portal Portal `json:"portal"`

	// If true, only websocket handshake is used, and tcp communication is used directly after the handshake is successful.
	Fast bool `json:"fast"`

	FS string `json:"fs"`
}
type BasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type ConnectDialer struct {
	// Connect dialer with tag 'tcp'
	Tag string `json:"tag"`
	// After one end of the connection is disconnected, wait for one second before closing the other end
	// (waiting for untransmitted data to continue transmitting)
	// Default "1s"
	Close string `json:"close"`
}
