package network

import (
	"container/list"
	"crypto/tls"
	"errors"
	"net"
	"runtime"

	"github.com/powerpuffpenguin/vnet"
)

type Network struct {
	pipe     map[string]*vnet.PipeListener
	pipeList *list.List
}

func New() *Network {
	return &Network{
		pipe:     make(map[string]*vnet.PipeListener),
		pipeList: list.New(),
	}
}
func (n *Network) listenPipe(address string) (l net.Listener, e error) {
	if _, ok := n.pipe[address]; ok {
		e = errors.New(`listen pipe ` + address + `: bind: address already in use`)
		return
	}
	pipe := vnet.ListenPipe()
	for ele := n.pipeList.Front(); ele != nil; ele = ele.Next() {
		d := ele.Value.(*pipeDialer)
		if d.addr == address {
			d.pipe = pipe
			close(d.done)
		}
	}

	l = pipe
	n.pipe[address] = pipe
	return
}
func (n *Network) Listen(network, address string) (l net.Listener, e error) {
	switch network {
	case `tcp`:
	case `pipe`:
		return n.listenPipe(address)
	case `unix`:
		if runtime.GOOS != `linux` {
			e = ErrNetworkUnix
			return
		}
	default:
		e = errors.New(`network not supported: ` + network)
		return
	}
	l, e = net.Listen(network, address)
	return
}
func (n *Network) ListenTLS(network, address string, config *tls.Config) (l net.Listener, e error) {
	if config == nil || len(config.Certificates) == 0 &&
		config.GetCertificate == nil && config.GetConfigForClient == nil {
		e = errors.New("tls: neither Certificates, GetCertificate, nor GetConfigForClient set in Config")
		return
	}
	l, e = n.Listen(network, address)
	if e != nil {
		return
	}
	l = tls.NewListener(l, config)
	return
}

func (n *Network) Dialer(network string, addr string, cfg *tls.Config) (dialer Dialer, e error) {
	switch network {
	case `pipe`:
		dialer = &pipeDialer{
			cfg:  cfg,
			addr: addr,
			done: make(chan struct{}),
		}
		n.pipeList.PushBack(dialer)
		return
	case `tcp`:
	case `unix`:
		if runtime.GOOS != `linux` {
			e = ErrNetworkUnix
			return
		}
	default:
		e = errors.New(`network not supported: ` + network)
		return
	}
	netDialer := &net.Dialer{}
	dialer = &rawDialer{
		netDialer: netDialer,
		network:   network,
		addr:      addr,
		cfg:       cfg,
	}
	return
}
