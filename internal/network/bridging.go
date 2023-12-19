package network

import (
	"errors"
	"io"
	"time"

	"github.com/powerpuffpenguin/streamf/pool"
	"github.com/powerpuffpenguin/streamf/third-party/websocket"
)

type websocketConn interface {
	Websocket() *websocket.Conn
}

func Bridging(c0, c1 io.ReadWriteCloser, pool *pool.Pool, duration time.Duration) {
	defer c0.Close()
	defer c1.Close()
	w0, ok0 := c0.(websocketConn)
	w1, ok1 := c1.(websocketConn)
	done := make(chan bool, 1)
	if ok0 {
		if ok1 {
			ws0 := w0.Websocket()
			ws1 := w1.Websocket()
			go forwardingWebsocket(ws0, ws1, done)
			go forwardingWebsocket(ws1, ws0, done)
		} else {
			go readWebsocket(c1, w0.Websocket(), done, pool)
			go writeWebsocket(w0.Websocket(), c1, done, pool)
		}
	} else if ok1 {
		go readWebsocket(c0, w1.Websocket(), done, pool)
		go writeWebsocket(w1.Websocket(), c0, done, pool)
	} else {
		go forwarding(c0, c1, done, pool)
		go forwarding(c1, c0, done, pool)
	}
	<-done
	if duration <= time.Millisecond {
		return
	}
	select {
	case <-done:
		return
	default:
	}

	if duration < time.Second*10 {
		time.Sleep(duration)
		return
	}
	timer := time.NewTimer(duration)
	select {
	case <-timer.C:
	case <-done:
		if !timer.Stop() {
			<-timer.C
		}
	}
}
func forwardingWebsocket(w, r *websocket.Conn, done chan<- bool) {
	defer forwardingDone(done)
	var (
		e   error
		t   int
		src io.Reader
		dst io.WriteCloser
	)
	for {
		t, src, e = r.NextReader()
		if e != nil {
			break
		}
		dst, e = w.NextWriter(t)
		if e != nil {
			break
		}
		_, e = io.Copy(dst, src)
		if e != nil {
			dst.Close()
			break
		}
		e = dst.Close()
		if e != nil {
			break
		}
	}
}

func readWebsocket(w io.WriteCloser, r *websocket.Conn, done chan<- bool, pool *pool.Pool) {
	defer forwardingDone(done)
	var (
		e   error
		src io.Reader
	)
	if rt, ok := w.(io.ReaderFrom); ok {
		for {
			_, src, e = r.NextReader()
			if e != nil {
				break
			}
			_, e = rt.ReadFrom(src)
			if e != nil {
				break
			}
		}
	} else {
		buf := pool.Get()
		for {
			_, src, e = r.NextReader()
			if e != nil {
				break
			}
			_, e = copyBuffer(w, src, buf)
			if e != nil {
				break
			}
		}
		pool.Put(buf)
	}
}
func writeWebsocket(w *websocket.Conn, r io.ReadCloser, done chan<- bool, pool *pool.Pool) {
	defer forwardingDone(done)
	var (
		b      = pool.Get()
		n      int
		er, ew error
	)
	for er == nil && ew == nil {
		n, er = r.Read(b)
		if n > 0 {
			ew = w.WriteMessage(websocket.BinaryMessage, b[:n])
		}
	}
	pool.Put(b)
}
func forwardingDone(done chan<- bool) {
	done <- true
}
func forwarding(w io.WriteCloser, r io.ReadCloser, done chan<- bool, pool *pool.Pool) {
	defer forwardingDone(done)
	// var (
	// 	b      = make([]byte, 32*1024)
	// 	n      int
	// 	er, ew error
	// )
	// for er == nil && ew == nil {
	// 	n, er = r.Read(b)
	// 	if n > 0 {
	// 		_, ew = w.Write(b[:n])
	// 	}
	// }
	if rt, ok := w.(io.ReaderFrom); ok {
		rt.ReadFrom(r)
	} else if wt, ok := r.(io.WriterTo); ok {
		wt.WriteTo(w)
	} else {
		var b = pool.Get()
		copyBuffer(w, r, b)
		pool.Put(b)
	}
}

// errInvalidWrite means that a write returned an impossible count.
var errInvalidWrite = errors.New("invalid write result")

// copyBuffer is the actual implementation of Copy and CopyBuffer.
// if buf is nil, one is allocated.
func copyBuffer(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	// // If the reader has a WriteTo method, use it to do the copy.
	// // Avoids an allocation and a copy.
	// if wt, ok := src.(WriterTo); ok {
	// 	return wt.WriteTo(dst)
	// }
	// // Similarly, if the writer has a ReadFrom method, use it to do the copy.
	// if rt, ok := dst.(ReaderFrom); ok {
	// 	return rt.ReadFrom(src)
	// }
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errInvalidWrite
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}
