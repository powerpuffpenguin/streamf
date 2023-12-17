package ioutil

import (
	"io"
	"net/http"
)

func NewReadWriter(r io.Reader, w io.Writer, c io.Closer) io.ReadWriteCloser {
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
	io.Reader
	w  io.Writer
	f  http.Flusher
	rt io.ReaderFrom
	c  io.Closer
}

func (rw *rwfReaderFrom) Write(b []byte) (int, error) {
	return rw.w.Write(b)
}
func (rw *rwfReaderFrom) Flush() {
	rw.f.Flush()
}
func (rw *rwfReaderFrom) ReadFrom(r io.Reader) (int64, error) {
	return rw.rt.ReadFrom(r)
}
func (rw *rwfReaderFrom) Close() error {
	return rw.c.Close()
}

type rwf struct {
	io.Reader
	w io.Writer
	f http.Flusher
	c io.Closer
}

func (rw *rwf) Write(b []byte) (int, error) {
	return rw.w.Write(b)
}
func (rw *rwf) Flush() {
	rw.f.Flush()
}
func (rw *rwf) Close() error {
	return rw.c.Close()
}

type rwReaderFrom struct {
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
	io.Reader
	io.Writer
	c io.Closer
}

func (rw *rw) Close() error {
	return rw.c.Close()
}
