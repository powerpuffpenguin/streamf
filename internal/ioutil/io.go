package ioutil

import (
	"io"
	"net"
	"net/http"
	"time"
)

type unknowAddr struct {
}

func (u unknowAddr) Network() string {
	return `unknow`
}
func (u unknowAddr) String() string {
	return `unknow`
}

type conn struct {
}

func (conn) LocalAddr() net.Addr {
	return unknowAddr{}
}
func (conn) RemoteAddr() net.Addr {
	return unknowAddr{}
}
func (conn) SetDeadline(t time.Time) error {
	return nil
}
func (conn) SetReadDeadline(t time.Time) error {
	return nil
}
func (conn) SetWriteDeadline(t time.Time) error {
	return nil
}

func NewReadWriter(r io.Reader, w io.Writer, c io.Closer) net.Conn {
	f, ok := w.(http.Flusher)
	if ok {
		if rt, ok := w.(io.ReaderFrom); ok {
			return &rwfReaderFrom{
				Reader: r,
				w:      w,
				f:      f,
				rt:     rt,
				c:      c,
			}
		}
		return &rwf{
			Reader: r,
			w:      w,
			f:      f,
			c:      c,
		}
	}
	if rt, ok := w.(io.ReaderFrom); ok {
		return &rwReaderFrom{
			Reader: r,
			w:      w,
			rt:     rt,
			c:      c,
		}
	}
	return &rw{
		Reader: r,
		Writer: w,
		c:      c,
	}
}

type rwfReaderFrom struct {
	conn
	io.Reader
	w  io.Writer
	f  http.Flusher
	rt io.ReaderFrom
	c  io.Closer
}

func (rw *rwfReaderFrom) Write(b []byte) (int, error) {
	n, e := rw.w.Write(b)
	if n > 0 {
		rw.f.Flush()
	}
	return n, e
}
func (rw *rwfReaderFrom) ReadFrom(r io.Reader) (int64, error) {
	return rw.rt.ReadFrom(r)
}
func (rw *rwfReaderFrom) Close() error {
	return rw.c.Close()
}

type rwf struct {
	conn
	io.Reader
	w io.Writer
	f http.Flusher
	c io.Closer
}

func (rw *rwf) Write(b []byte) (int, error) {
	n, e := rw.w.Write(b)
	if n > 0 {
		rw.f.Flush()
	}
	return n, e
}

func (rw *rwf) Close() error {
	return rw.c.Close()
}

type rwReaderFrom struct {
	conn
	io.Reader
	w  io.Writer
	rt io.ReaderFrom
	c  io.Closer
}

func (rw *rwReaderFrom) Write(b []byte) (int, error) {
	return rw.w.Write(b)
}

func (rw *rwReaderFrom) ReadFrom(r io.Reader) (int64, error) {
	return rw.rt.ReadFrom(r)
}
func (rw *rwReaderFrom) Close() error {
	return rw.c.Close()
}

type rw struct {
	conn
	io.Reader
	io.Writer
	c io.Closer
}

func (rw *rw) Close() error {
	return rw.c.Close()
}
