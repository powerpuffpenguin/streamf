package network

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
)

var errUdpListenerClosed = errors.New(`udp listener already closed`)
var errUdpConnClosed = errors.New(`udp conn already closed`)

type udpListener struct {
	size    int
	timeout time.Duration
	addr    *net.UDPAddr

	l *net.UDPConn

	closed uint32
	done   chan struct{}
	msg    chan readUdpMessage
	ch     chan *udpToTcp

	close chan *udpToTcp
}

func newUdpListener(address string, opts *config.UDP) (l *udpListener, e error) {
	addr, e := net.ResolveUDPAddr(`udp`, address)
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
	ul, e := net.ListenUDP("udp", addr)
	if e != nil {
		return
	}
	l = &udpListener{
		addr: addr,

		l:       ul,
		done:    make(chan struct{}),
		msg:     make(chan readUdpMessage, 32),
		ch:      make(chan *udpToTcp),
		close:   make(chan *udpToTcp),
		size:    size,
		timeout: timeout,
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
		e    error
		n    int
		addr *net.UDPAddr
	)
	for {
		b = make([]byte, 2048+2)
		n, addr, e = u.l.ReadFromUDP(b[2:])
		if e != nil {
			select {
			case <-u.done:
				return
			default:
			}
			continue
		}
		select {
		case <-u.done:
			return
		case u.msg <- readUdpMessage{
			addr: addr,
			b:    b[:n+2],
		}:
		}
	}
}

type readUdpMessage struct {
	addr *net.UDPAddr
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
				c.putRead(msg.b)
			} else {
				c = newUdpToTcp(u, msg.addr)
				select {
				case <-u.done:
					return
				case u.ch <- c:
					keys[key] = c
					c.putRead(msg.b)
				}
			}
		}
	}
}

type udpToTcp struct {
	l    *udpListener
	addr *net.UDPAddr

	closed uint32
	done   chan struct{}

	ch     chan []byte
	read   []byte
	signal chan bool

	r *io.PipeReader
	w *io.PipeWriter
}

func newUdpToTcp(l *udpListener, addr *net.UDPAddr) *udpToTcp {
	r, w := io.Pipe()
	c := &udpToTcp{
		l:      l,
		addr:   addr,
		done:   make(chan struct{}),
		ch:     make(chan []byte, 16),
		signal: make(chan bool, 1),
		r:      r,
		w:      w,
	}
	go c.run()
	return c
}

func (c *udpToTcp) putRead(msg []byte) {
	select {
	case c.signal <- true:
	default:
	}

	binary.LittleEndian.PutUint16(msg, uint16(len(msg)-2))
	for {
		select {
		case <-c.done:
			return
		case c.ch <- msg:
			return
		default:
		}

		select {
		case <-c.done:
			return
		case c.ch <- msg:
			return
		case <-c.ch:
			fmt.Println(`loss`, time.Now(), len(msg)-2)
		}
	}

}

func (c *udpToTcp) Read(b []byte) (n int, err error) {
	for {
		if len(c.read) > 0 {
			n = copy(b, c.read)
			c.read = c.read[n:]
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
func (c *udpToTcp) run() {
	go func() {
		timer := time.NewTicker(time.Second * 10)
		count := 0
		for {
			select {
			case <-timer.C:
				count++
				if count == 6 { //timeout
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
		b = make([]byte, 1024*2)
		e error
		n int
	)
	for {
		_, e = io.ReadAtLeast(c.r, b[:2], 2)
		if e != nil {
			break
		}
		n = int(binary.LittleEndian.Uint16(b))
		if n > len(b) {
			b = make([]byte, n)
		}
		if n > 0 {
			_, e = io.ReadAtLeast(c.r, b[:n], n)
			if e != nil {
				break
			}
			_, e = c.l.l.WriteToUDP(b[:n], c.addr)
			if e != nil {
				fmt.Println(`Write udp err`, e)
				break
			}
			select {
			case c.signal <- true:
			default:
			}
		}
	}
	c.Close()
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
