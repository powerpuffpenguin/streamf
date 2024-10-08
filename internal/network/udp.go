package network

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/pool"
)

var errUdpListenerClosed = errors.New(`udp listener already closed`)
var errUdpConnClosed = errors.New(`udp conn already closed`)

type udpListener struct {
	size, frame int
	timeout     time.Duration
	pool        *pool.Pool
	addr        *net.UDPAddr

	l *net.UDPConn

	closed uint32
	done   chan struct{}
	msg    chan readUdpMessage
	ch     chan *udpToTcp

	close chan *udpToTcp
}

func newUdpListener(network, address string, opts *config.UDP, pool *pool.Pool) (l *udpListener, e error) {
	addr, e := net.ResolveUDPAddr(network, address)
	if e != nil {
		return
	}
	var timeout time.Duration
	if opts.Timeout == `` {
		timeout = time.Second * 60
	} else {
		var err error
		timeout, err = time.ParseDuration(opts.Timeout)
		if err != nil {
			timeout = time.Second * 60
		}
	}
	size := opts.Size
	if size < 128 {
		size = 1024 * 2
	}
	frame := opts.Frame
	if frame < 1 {
		frame = 16
	}
	ul, e := net.ListenUDP(network, addr)
	if e != nil {
		return
	}
	l = &udpListener{
		addr: addr,

		l:       ul,
		done:    make(chan struct{}),
		msg:     make(chan readUdpMessage, frame),
		ch:      make(chan *udpToTcp),
		close:   make(chan *udpToTcp),
		size:    size,
		timeout: timeout,
		frame:   frame,
		pool:    pool,
	}
	go l.run()
	return
}

func (u *udpListener) Accept() (c net.Conn, e error) {
	select {
	case <-u.done:
		e = errUdpListenerClosed
	case uc := <-u.ch:
		c = uc
	}
	return
}

func (u *udpListener) Close() (e error) {
	if u.closed == 0 && atomic.CompareAndSwapUint32(&u.closed, 0, 1) {
		close(u.done)
		e = u.l.Close()
	} else {
		e = errUdpListenerClosed
	}
	return
}

func (u *udpListener) Addr() net.Addr {
	return u.addr
}
func (u *udpListener) run() {
	go u.onMessage()
	var (
		b    []byte
		data []byte
		e    error
		n    int
		addr *net.UDPAddr
		size = u.size + 2
	)
	for {
		if u.pool.Size() >= size {
			data = u.pool.Get()
			b = data
		} else {
			data = nil
			b = make([]byte, size)
		}
		n, addr, e = u.l.ReadFromUDP(b[2:])
		if e != nil {
			if data != nil {
				u.pool.Put(data)
			}
			select {
			case <-u.done:
				return
			default:
			}
			continue
		}
		select {
		case <-u.done:
			if data != nil {
				u.pool.Put(data)
			}
			return
		case u.msg <- readUdpMessage{
			addr: addr,
			b:    b[:n+2],
			data: data,
		}:
		}
	}
}

type readUdpMessage struct {
	addr *net.UDPAddr
	data []byte
	b    []byte
}

func (u *udpListener) onMessage() {
	var (
		msg   readUdpMessage
		keys  = make(map[string]*udpToTcp)
		key   string
		ok    bool
		c, c0 *udpToTcp
	)
	for {
		select {
		case <-u.done:
			return
		case c = <-u.close:
			key = c.addr.String()
			if c0, ok = keys[key]; ok && c == c0 {
				delete(keys, key)
			}
		case msg = <-u.msg:
			key = msg.addr.String()
			if c, ok = keys[key]; ok {
				c.putRead(msg)
			} else {
				c = newUdpToTcp(u, msg.addr, u.pool, u.timeout, u.size, u.frame)
				select {
				case <-u.done:
					if msg.data != nil {
						u.pool.Put(msg.data)
					}
					return
				case u.ch <- c:
					keys[key] = c
					c.putRead(msg)
				}
			}
		}
	}
}

type udpToTcp struct {
	pool *pool.Pool
	size int
	l    *udpListener
	addr *net.UDPAddr

	closed uint32
	done   chan struct{}

	ch     chan readUdpMessage
	read   readUdpMessage
	signal chan bool

	r *io.PipeReader
	w *io.PipeWriter
}

func newUdpToTcp(l *udpListener, addr *net.UDPAddr, pool *pool.Pool, timeout time.Duration, size, frame int) *udpToTcp {
	r, w := io.Pipe()
	c := &udpToTcp{
		pool:   pool,
		size:   size,
		l:      l,
		addr:   addr,
		done:   make(chan struct{}),
		ch:     make(chan readUdpMessage, frame),
		signal: make(chan bool, 1),
		r:      r,
		w:      w,
	}
	go c.run(timeout)
	return c
}

func (c *udpToTcp) putRead(msg readUdpMessage) {
	select {
	case c.signal <- true:
	default:
	}

	binary.LittleEndian.PutUint16(msg.b, uint16(len(msg.b)-2))
	var v readUdpMessage
	for {
		select {
		case <-c.done:
			if msg.data != nil {
				c.pool.Put(msg.data)
			}
			return
		case c.ch <- msg:
			return
		default:
		}

		select {
		case <-c.done:
			if msg.data != nil {
				c.pool.Put(msg.data)
			}
			return
		case c.ch <- msg:
			return
		case v = <-c.ch:
			if v.data != nil {
				c.pool.Put(v.data)
			}
		}
	}

}

func (c *udpToTcp) Read(b []byte) (n int, err error) {
	for {
		if len(c.read.b) > 0 {
			n = copy(b, c.read.b)
			c.read.b = c.read.b[n:]
			if len(c.read.b) == 0 && c.read.data != nil {
				c.pool.Put(c.read.data)
				c.read.data = nil
			}
			break
		}
		select {
		case <-c.done:
			err = errUdpConnClosed
			return
		case <-c.l.done:
			err = io.EOF
			return
		case c.read = <-c.ch:
		}
	}
	return
}

func (c *udpToTcp) Write(b []byte) (n int, err error) {
	return c.w.Write(b)
}
func (c *udpToTcp) run(timeout time.Duration) {
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
			_, e = c.l.l.WriteToUDP(b[:n], c.addr)
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
func (c *udpToTcp) Close() (e error) {
	if c.closed == 0 && atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		close(c.done)
		c.r.Close()
		c.w.Close()
		select {
		case <-c.l.done:
		case c.l.close <- c:
		}
	} else {
		e = errUdpConnClosed
	}
	return e
}

func (c *udpToTcp) LocalAddr() net.Addr {
	return c.l.l.LocalAddr()
}

func (c *udpToTcp) RemoteAddr() net.Addr {
	return c.addr
}
func (c *udpToTcp) SetDeadline(t time.Time) error {
	return nil
}
func (c *udpToTcp) SetReadDeadline(t time.Time) error {
	return nil
}
func (c *udpToTcp) SetWriteDeadline(t time.Time) error {
	return nil
}
