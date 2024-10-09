package udp

import (
	"errors"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
)

type UDP struct {
	tag       string
	listen    string
	c         *net.UDPConn
	toNetwork string
	to        string

	timeout time.Duration
	size    int
	mutex   sync.Mutex
	keys    map[string]*remoteConn

	done   chan struct{}
	closed uint32

	log *slog.Logger
}

func New(log *slog.Logger, opts *config.UDPForward) (u *UDP, e error) {
	tag := opts.Tag
	network := opts.Network
	if network == `` {
		network = `udp`
	} else if network != `udp` && network != `udp4` && network != `udp6` {
		e = errors.New(`network not supported: ` + network)
		return
	}
	toNetwork := opts.ToNetwork
	if toNetwork == `` {
		toNetwork = `udp`
	} else if toNetwork != `udp` && toNetwork != `udp4` && toNetwork != `udp6` {
		e = errors.New(`network not supported: ` + toNetwork)
		return
	}
	if tag == `` {
		tag = network + ` ` + opts.Listen + ` -> ` + toNetwork + ` ` + opts.To
	}
	log = log.With(
		`tag`, tag,
		`listener`, opts.Listen,
		`to`, opts.To)
	addr, e := net.ResolveUDPAddr(network, opts.Listen)
	if e != nil {
		log.Error(`listen udp fial`, `error`, e)
		return
	}
	c, e := net.ListenUDP(network, addr)
	if e != nil {
		log.Error(`listen udp fial`, `error`, e)
		return
	}
	var timeout time.Duration
	if opts.Timeout == `` {
		timeout = time.Minute * 3
	} else {
		var err error
		timeout, err = time.ParseDuration(opts.Timeout)
		if err != nil {
			timeout = time.Minute * 3
		}
	}
	size := opts.Size
	if size < 128 {
		size = 1024 * 2
	}
	log.Info(`udp forward`, `timeout`, timeout, `size`, size)
	u = &UDP{
		tag:       tag,
		c:         c,
		listen:    opts.Listen,
		to:        opts.To,
		toNetwork: opts.ToNetwork,
		timeout:   timeout,
		size:      size,
		keys:      make(map[string]*remoteConn),
		done:      make(chan struct{}),
		log:       log,
	}
	return
}
func (u *UDP) Info() any {
	return map[string]any{
		`tag`:     u.tag,
		`listen`:  u.listen,
		`to`:      u.to,
		`timeout`: u.timeout.String(),
		`size`:    u.size,
	}
}
func (u *UDP) Serve() (e error) {
	var (
		b    = make([]byte, u.size)
		n    int
		addr *net.UDPAddr
		key  string
		c    *remoteConn
		ok   bool
		conn *net.UDPConn
		to   *net.UDPAddr
	)
	for {
		n, addr, e = u.c.ReadFromUDP(b)
		if e != nil {
			u.log.Warn("ReadFromUDP fail", `error`, e)
			break
		}
		key = addr.String()
		if c, ok = u.keys[key]; ok {
			_, e = c.Write(b[:n])
			if e != nil {
				u.log.Warn("UDP Write fail", `error`, e)
				continue
			}
		} else {
			to, e = net.ResolveUDPAddr(u.toNetwork, u.to)
			if e != nil {
				u.log.Warn("ResolveUDPAddr fail", `error`, e)
				continue
			}
			conn, e = net.DialUDP(u.toNetwork, nil, to)
			if e != nil {
				u.log.Warn("DialUDP fail", `error`, e)
				continue
			}
			_, e = conn.Write(b[:n])
			if e != nil {
				u.log.Warn("UDP Write fail", `error`, e)
				conn.Close()
				continue
			}
			c = newRemoteConn(u, conn, key, addr)
			u.mutex.Lock()
			u.keys[key] = c
			u.mutex.Unlock()
		}
	}
	return
}
func (u *UDP) Close() (e error) {
	if u.closed == 0 && atomic.CompareAndSwapUint32(&u.closed, 0, 1) {
		close(u.done)
		e = u.c.Close()
	}
	return
}

type remoteConn struct {
	udp    *UDP
	c      *net.UDPConn
	key    string
	addr   *net.UDPAddr
	done   chan struct{}
	closed uint32

	ch     chan bool
	ticker *time.Ticker
}

func newRemoteConn(udp *UDP, conn *net.UDPConn, key string, addr *net.UDPAddr) (c *remoteConn) {
	c = &remoteConn{
		udp:  udp,
		c:    conn,
		key:  key,
		addr: addr,
		done: make(chan struct{}),
	}
	if udp.timeout > time.Second {
		c.ch = make(chan bool)
		c.ticker = time.NewTicker(time.Second * 10)
		go func() {
			max := udp.timeout / time.Second * 10
			var n time.Duration = 0
			for {
				select {
				case <-udp.done:
					c.Close()
					return
				case <-c.done:
					return
				case <-c.ticker.C:
					if n >= max {
						c.Close()
						return
					} else {
						n++
					}
				case <-c.ch:
					n = 0
				}
			}
		}()
	}
	go c.run()
	return
}
func (c *remoteConn) run() {
	defer c.Close()
	var (
		b = make([]byte, c.udp.size)
		e error
		n int
	)
	for {
		n, e = c.c.Read(b)
		if e != nil {
			break
		}
		if c.ch != nil {
			select {
			case <-c.done:
				return
			case <-c.udp.done:
				return
			case c.ch <- true:
			default:
			}
		}
		if n == 0 {
			continue
		}
		_, e = c.udp.c.WriteToUDP(b[:n], c.addr)
		if e != nil {
			break
		}
	}
}
func (c *remoteConn) Write(b []byte) (n int, e error) {
	n, e = c.c.Write(b)
	if e == nil {
		if c.ch != nil {
			select {
			case <-c.done:
			case <-c.udp.done:
			case c.ch <- true:
			default:
			}
		}
	} else {
		c.Close()
	}
	return
}
func (c *remoteConn) Close() (e error) {
	if c.closed == 0 && atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		close(c.done)
		e = c.c.Close()
		if c.ticker != nil {
			c.ticker.Stop()
		}

		c.udp.mutex.Lock()
		if c.udp.keys[c.key] == c {
			delete(c.udp.keys, c.key)
		}
		c.udp.mutex.Unlock()
	}
	return
}
