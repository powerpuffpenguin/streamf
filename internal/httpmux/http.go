package httpmux

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/powerpuffpenguin/streamf/internal/ioutil"
)

type connectResult struct {
	conn net.Conn
	e    error
}

func ConnectHttp(ctx context.Context, client *http.Client, method, url string, header http.Header) (conn net.Conn, e error) {
	newctx, cancel := context.WithCancel(context.Background())
	ch := make(chan connectResult)
	go func() {
		conn, e := connectHttp(newctx, cancel, client, method, url, header)
		select {
		case <-ctx.Done():
			if e == nil {
				conn.Close()
			}
		case ch <- connectResult{
			conn: conn,
			e:    e,
		}:
		}
	}()
	select {
	case <-ctx.Done():
		e = ctx.Err()
		cancel()
	case result := <-ch:
		conn = result.conn
		e = result.e
		if e != nil {
			cancel()
		}
	}
	return
}
func connectHttp(ctx context.Context, cancel context.CancelFunc, client *http.Client, method, url string, header http.Header) (conn net.Conn, e error) {
	r, w := io.Pipe()
	req, e := http.NewRequestWithContext(ctx, method, url, r)
	if e != nil {
		w.Close()
		return
	}
	req.Header = header
	resp, e := client.Do(req)
	if e != nil {
		w.Close()
		return
	} else if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusAccepted {
		defer w.Close()
		var b []byte
		if resp.Body != nil {
			b, _ = io.ReadAll(io.LimitReader(resp.Body, 1024))
			resp.Body.Close()
		}
		if len(b) == 0 {
			e = errors.New(resp.Status)
		} else {
			e = errors.New(resp.Status + ` ` + string(b))
		}
		return
	} else if resp.Body == nil {
		e = errors.New(`http body nil`)
		return
	}
	conn = ioutil.NewReadWriter(resp.Body, w, &httpCloser{
		cancel: cancel,
		w:      w,
		r:      r,
	})
	return
}

type httpCloser struct {
	w      *io.PipeWriter
	r      io.ReadCloser
	closed uint32
	cancel context.CancelFunc
}

func (c *httpCloser) Close() (e error) {
	if c.closed == 0 && atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		c.w.Close()
		c.r.Close()
		c.cancel()
	} else {
		e = ErrDialerClosed
	}
	return
}

var ErrDialerClosed = errors.New(`dialer already closed`)
