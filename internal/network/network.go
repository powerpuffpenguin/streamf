package network

import (
	"container/list"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"runtime"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/vnet"
	"github.com/powerpuffpenguin/vnet/reverse"
)

type Network struct {
	pipe     map[string]*vnet.PipeListener
	pipeList *list.List

	portal     map[string]*reverse.Dialer
	portalList *list.List
}

func New() *Network {
	return &Network{
		pipe:       make(map[string]*vnet.PipeListener),
		pipeList:   list.New(),
		portal:     make(map[string]*reverse.Dialer),
		portalList: list.New(),
	}
}
func (n *Network) listenPipe(address string) (l net.Listener, e error) {
	if _, ok := n.pipe[address]; ok {
		e = errors.New(`listen pipe ` + address + `: bind: address already in use`)
		return
	}
	var (
		pipe = vnet.ListenPipe()
		ele  = n.pipeList.Front()
		next *list.Element
	)
	for ele != nil {
		next = ele.Next()
		d := ele.Value.(*pipeDialer)
		if d.addr == address {
			d.pipe = pipe
			close(d.done)
			n.pipeList.Remove(ele)
		}
		ele = next
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
	case `portal`:
		dialer = &portalDialer{
			cfg:  cfg,
			addr: addr,
			done: make(chan struct{}),
		}
		n.portalList.PushBack(dialer)
		return
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
func (n *Network) NewPortal(log *slog.Logger, l net.Listener, portal *config.Portal) (dialer *reverse.Dialer, e error) {
	tag := portal.Tag
	if _, ok := n.portal[tag]; ok {
		e = errors.New(`portal already exists: ` + tag)
		return
	}
	var timeout time.Duration
	if portal.Timeout == `` {
		timeout = time.Millisecond * 500
	} else {
		var err error
		timeout, err = time.ParseDuration(portal.Timeout)
		if err != nil {
			timeout = time.Millisecond * 500
			log.Warn(`parse duration fail, used default timeout duration.`,
				`error`, err,
				`timeout`, portal.Timeout,
				`default`, timeout,
			)
		}
	}
	var heart time.Duration
	if portal.Heart == `` {
		heart = time.Second * 40
	} else {
		var err error
		heart, err = time.ParseDuration(portal.Heart)
		if err != nil {
			heart = time.Second * 40
			log.Warn(`parse duration fail, used default heart duration.`,
				`error`, err,
				`heart`, portal.Heart,
				`default`, timeout,
			)
		}
	}
	var heartTimeout time.Duration
	if portal.HeartTimeout == `` {
		heartTimeout = time.Second * 1
	} else {
		var err error
		heartTimeout, err = time.ParseDuration(portal.HeartTimeout)
		if err != nil {
			heartTimeout = time.Second * 1
			log.Warn(`parse duration fail, used default heartTimeout duration.`,
				`error`, err,
				`heartTimeout`, portal.HeartTimeout,
				`default`, heartTimeout,
			)
		}
	}
	dialer = reverse.NewDialer(l,
		reverse.WithDialerSynAck(true),
		reverse.WithDialerTimeout(timeout),
		reverse.WithDialerHeart(heart),
	)
	n.portal[tag] = dialer
	log.Info(`new portal listener`,
		`timeout`, timeout,
		`heart`, heart,
		`heartTimeout`, heartTimeout,
	)

	var (
		ele  = n.portalList.Front()
		next *list.Element
	)
	for ele != nil {
		next = ele.Next()
		d := ele.Value.(*portalDialer)
		if d.addr == tag {
			d.portal = dialer
			close(d.done)
			n.portalList.Remove(ele)
		}
		ele = next
	}
	return
}
