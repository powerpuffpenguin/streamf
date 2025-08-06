package config

type SNIProxy struct {
	// Custom name recorded in logs
	Tag     string `json:"tag"`
	Network string `json:"network"`
	Addr    string `json:"addr"`
	// Sniff sni timeout, Default 500ms
	Timeout string `json:"timeout"`

	SNIRouter []*SNIRouter `json:"router"`
}
type SNIRouter struct {
	Matcher []SNIMatcher `json:"matcher"`
	// Specify forwarding destination
	Dialer ConnectDialer `json:"dialer"`
}
type SNIMatcher struct {
	// 'equal' 'prefix' 'suffix' 'regexp'
	Type  string `json:"type"`
	Value string `json:"value"`
}
