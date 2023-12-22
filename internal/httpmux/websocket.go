package httpmux

import (
	"io"
	"net"
	"time"

	"github.com/powerpuffpenguin/streamf/third-party/websocket"
)

type WebsocketConn struct {
	ws *websocket.Conn
	r  io.Reader
}

func NewWebsocketConn(ws *websocket.Conn) *WebsocketConn {
	return &WebsocketConn{ws: ws}
}

func (w *WebsocketConn) Websocket() *websocket.Conn {
	return w.ws
}
func (w *WebsocketConn) Close() error {
	return w.ws.Close()
}
func (w *WebsocketConn) Write(b []byte) (n int, e error) {
	e = w.ws.WriteMessage(websocket.BinaryMessage, b)
	if e == nil {
		n = len(b)
	}
	return
}
func (w *WebsocketConn) Read(b []byte) (n int, e error) {
	r := w.r
	if r == nil {
		_, r, e = w.ws.NextReader()
		if e != nil {
			return
		}
		w.r = r
	}
	n, e = w.r.Read(b)
	if e == io.EOF {
		e = nil
		w.r = nil
	}
	return
}

func (w *WebsocketConn) LocalAddr() net.Addr  { return w.ws.LocalAddr() }
func (w *WebsocketConn) RemoteAddr() net.Addr { return w.ws.RemoteAddr() }

func (w *WebsocketConn) SetDeadline(t time.Time) error {
	e := w.ws.SetReadDeadline(t)
	if e != nil {
		return e
	}
	e = w.ws.SetWriteDeadline(t)
	return e
}

func (w *WebsocketConn) SetReadDeadline(t time.Time) error {
	return w.ws.SetReadDeadline(t)
}

func (w *WebsocketConn) SetWriteDeadline(t time.Time) error {
	return w.ws.SetWriteDeadline(t)
}
