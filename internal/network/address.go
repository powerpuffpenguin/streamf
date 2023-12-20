package network

import "net"

type address struct {
	network string
	addr    string
}

func NewAddr(network, addr string) net.Addr {
	return address{
		network: network,
		addr:    addr,
	}
}
func (a address) Network() string {
	return a.network
}
func (a address) String() string {
	return a.addr
}
