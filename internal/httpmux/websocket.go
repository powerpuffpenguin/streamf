package httpmux

import (
	"io"

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
	// panic(`not implemented websocketConn.Write`)
	e = w.ws.WriteMessage(websocket.BinaryMessage, b)
	if e != nil {
		n = len(b)
	}
	return
}
func (w *WebsocketConn) Read(b []byte) (n int, e error) {
	// panic(`not implemented websocketConn.Read`)
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
