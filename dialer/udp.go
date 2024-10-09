package dialer

import (
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"github.com/powerpuffpenguin/streamf/pool"
)

type udpDialer struct {
	url        string
	remoteAddr RemoteAddr
	timeout    time.Duration
	pool       *pool.Pool
	size       int
	frame      int
}

func newUdpDialer(
	nk *network.Network, log *slog.Logger,
	opts *config.Dialer, u *url.URL,
	pool *pool.Pool) (dialer *udpDialer, e error) {
	log = log.With(`dialer`, opts.Tag)
	var timeout time.Duration
	if opts.UDP.Timeout == `` {
		timeout = time.Second * 60
	} else {
		var err error
		timeout, err = time.ParseDuration(opts.UDP.Timeout)
		if err != nil {
			timeout = time.Second * 60
			log.Warn(`parse duration fail, used default timeout duration.`,
				`error`, err,
				`timeout`, timeout,
			)
		}
	}
	size := opts.UDP.Size
	if size < 128 {
		size = 1024 * 2
	}
	frame := opts.UDP.Frame
	if frame < 1 {
		frame = 16
	}

	var (
		network = `udp`
		addr    = u.Host
		query   url.Values
	)
	if opts.Network != `` {
		network = opts.Network
	} else {
		query = u.Query()
		s := query.Get(`network`)
		if s != `` {
			network = s
		}
	}
	if opts.Addr != `` {
		addr = opts.Addr
	} else {
		if query == nil {
			query = u.Query()
		}
		s := query.Get(`addr`)
		if s != `` {
			addr = s
		}
	}
	if e != nil {
		log.Error(`new dialer fail`, `error`, e)
		return
	}
	log.Info(`new dialer`,
		`network`, network,
		`addr`, addr,
		`url`, opts.URL,
		`timeout`, timeout,
		`size`, size,
		`frame`, frame,
	)
	dialer = &udpDialer{
		remoteAddr: RemoteAddr{
			Dialer:  opts.Tag,
			Network: network,
			Addr:    addr,
			Secure:  false,
			URL:     opts.URL,
		},
		url:     opts.URL,
		timeout: timeout,
		pool:    pool,
		size:    size,
		frame:   frame,
	}
	return
}
func (u *udpDialer) Tag() string {
	return u.remoteAddr.Dialer
}
func (u *udpDialer) Connect(ctx context.Context) (conn *Conn, e error) {
	addr, e := net.ResolveUDPAddr(u.remoteAddr.Network, u.remoteAddr.Addr)
	if e != nil {
		return
	}
	c, e := net.DialUDP(u.remoteAddr.Network, nil, addr)
	if e != nil {
		return
	}
	conn = &Conn{
		ReadWriteCloser: newTcpFromUdp(c, u.pool, u.timeout, u.size, u.frame),
		remoteAddr:      u.remoteAddr,
	}
	return
}
func (u *udpDialer) Close() (e error) {
	return nil
}
func (u *udpDialer) Info() any {
	return map[string]any{
		`tag`:     u.remoteAddr.Dialer,
		`network`: u.remoteAddr.Network,
		`addr`:    u.remoteAddr.Addr,
		`url`:     u.remoteAddr.URL,
		`secure`:  u.remoteAddr.Secure,
		`timeout`: u.timeout.String(),
		`size`:    u.size,
		`frame`:   u.frame,
	}
}

type tcpFromUdp struct {
	pool *pool.Pool
	size int

	c     *net.UDPConn
	close uint32
	done  chan struct{}

	ch     chan []byte
	buf    []byte
	read   []byte
	signal chan bool

	r *io.PipeReader
	w *io.PipeWriter
}

func newTcpFromUdp(c *net.UDPConn, pool *pool.Pool, timeout time.Duration, size, frame int) *tcpFromUdp {
	r, w := io.Pipe()
	cc := &tcpFromUdp{
		pool: pool,
		size: size,

		c:    c,
		done: make(chan struct{}),

		ch:     make(chan []byte, frame),
		buf:    make([]byte, size+2),
		signal: make(chan bool, 1),
		r:      r,
		w:      w,
	}
	go cc.run(timeout)
	return cc
}
func (c *tcpFromUdp) Read(b []byte) (n int, e error) {
	if len(b) == 0 {
		select {
		case <-c.done:
			e = io.EOF
		default:
		}
		return
	}

	var i int
	for {
		if len(c.read) > 0 {
			n = copy(b, c.read)
			c.read = c.read[n:]
			break
		}
		i, e = c.c.Read(c.buf[2:])
		if e != nil {
			return
		} else if i > 0 {
			binary.LittleEndian.PutUint16(c.buf, uint16(i))
			c.read = c.buf[:2+i]
		}
	}
	return
}
func (c *tcpFromUdp) Write(b []byte) (n int, e error) {
	return c.w.Write(b)
}
func (c *tcpFromUdp) run(timeout time.Duration) {
	go func() {
		var (
			wait  = timeout / (time.Second * 10)
			timer = time.NewTicker(time.Second * 10)
			count time.Duration
		)
		if wait == 0 {
			wait = 1
		}
		for {
			select {
			case <-timer.C:
				count++
				if count == wait { //timeout
					timer.Stop()
					c.Close()
					return
				}
			case <-c.signal:
				count = 0
			}
		}
	}()

	var (
		data []byte
		b    []byte
		e    error
		n    int
	)
	if c.pool.Size() < c.size {
		b = make([]byte, c.size)
	} else {
		data = c.pool.Get()
		b = data
	}

	for {
		_, e = io.ReadAtLeast(c.r, b[:2], 2)
		if e != nil {
			break
		}
		n = int(binary.LittleEndian.Uint16(b))
		if n > len(b) {
			b = make([]byte, n)
			if data != nil {
				c.pool.Put(data)
				data = nil
			}
		}
		if n > 0 {
			_, e = io.ReadAtLeast(c.r, b[:n], n)
			if e != nil {
				break
			}
			_, e = c.c.Write(b[:n])
			if e != nil {
				break
			}
			select {
			case c.signal <- true:
			default:
			}
		}
	}
	c.Close()
	if data != nil {
		c.pool.Put(data)
	}
}
func (c *tcpFromUdp) Close() (e error) {
	if c.close == 0 && atomic.CompareAndSwapUint32(&c.close, 0, 1) {
		close(c.done)

		c.r.Close()
		c.w.Close()
		e = c.c.Close()
	}
	return
}
