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

func ConnectHttp(ctx context.Context, client *http.Client, method, url string, header http.Header) (conn net.Conn, e error) {
	r, w := io.Pipe()
	req, e := http.NewRequest(method, url, r)
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
		w: w,
		r: r,
	})
	return
}

type httpCloser struct {
	w      *io.PipeWriter
	r      io.ReadCloser
	closed uint32
}

func (c *httpCloser) Close() (e error) {
	if c.closed == 0 && atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		c.w.Close()
		c.r.Close()
	} else {
		e = ErrDialerClosed
	}
	return
}

var ErrDialerClosed = errors.New(`dialer already closed`)
