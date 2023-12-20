package listener

import (
	"net"
	"sync/atomic"
)

type httpListener struct {
	done     <-chan struct{}
	selfDone chan struct{}
	closed   uint32
	addr     net.Addr
	ch       chan net.Conn
}

func newHttpListener(done <-chan struct{}, addr net.Addr) *httpListener {
	return &httpListener{
		done:     done,
		selfDone: make(chan struct{}),
		addr:     addr,
		ch:       make(chan net.Conn),
	}
}

// Accept waits for and returns the next connection to the listener.
func (l *httpListener) Accept() (c net.Conn, e error) {
	select {
	case <-l.done:
		e = ErrClosed
	case <-l.selfDone:
		e = ErrClosed
	case c = <-l.ch:
	}
	return
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *httpListener) Close() (e error) {
	if l.closed == 0 && atomic.CompareAndSwapUint32(&l.closed, 0, 1) {
		close(l.selfDone)
	}
	return
}

// Addr returns the listener's network address.
func (l *httpListener) Addr() net.Addr {
	return l.addr
}
