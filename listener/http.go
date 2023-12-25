package listener

import (
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/dialer"
	"github.com/powerpuffpenguin/streamf/internal/httpmux"
	"github.com/powerpuffpenguin/streamf/internal/ioutil"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"github.com/powerpuffpenguin/streamf/pool"
	"github.com/powerpuffpenguin/streamf/third-party/websocket"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type HttpListener struct {
	done              chan struct{}
	server            http.Server
	listener          net.Listener
	certFile, keyFile string
	pool              *pool.Pool
	log               *slog.Logger
	closed            uint32
	upgrader          *websocket.Upgrader

	closer []io.Closer

	tag, network, addr string
	secure             bool
}

func (l *HttpListener) Info() any {
	return map[string]any{
		`tag`:     l.tag,
		`network`: l.network,
		`addr`:    l.addr,
		`secure`:  l.secure,
		`portal`:  true,
	}
}
func NewHttpListener(nk *network.Network,
	log *slog.Logger, pool *pool.Pool,
	dialers map[string]dialer.Dialer,
	api []httpmux.ApiHandler,
	opts *config.BasicListener, routers []*config.Router,
) (listener *HttpListener, e error) {
	var (
		l      net.Listener
		secure bool
	)
	if opts.TLS.CertFile != `` && opts.TLS.KeyFile != `` {
		secure = true
	}
	l, e = nk.Listen(opts.Network, opts.Addr)
	if e != nil {
		log.Error(`new http listener fail`, `error`, e)
		return
	}
	addr := l.Addr()
	tag := opts.Tag
	if tag == `` {
		if secure {
			tag = `https ` + addr.Network() + `+tls://` + addr.String()
		} else {
			tag = `http ` + addr.Network() + `://` + addr.String()
		}
	}
	log = log.With(`listener`, tag)

	log.Info(`new http listener`,
		`network`, addr.Network(),
		`addr`, addr.String(),
	)
	listener = &HttpListener{
		done:     make(chan struct{}),
		certFile: opts.TLS.CertFile,
		keyFile:  opts.TLS.KeyFile,
		listener: l,
		pool:     pool,
		log:      log,

		tag:     tag,
		network: addr.Network(),
		addr:    addr.String(),
		secure:  secure,
	}
	var (
		mux     = httpmux.New(log)
		handler http.HandlerFunc
	)
	for _, router := range routers {
		switch strings.ToUpper(router.Method) {
		case ``, http.MethodPost:
			if router.Portal.Tag == `` {
				handler, e = listener.createHttp2(dialers, router)
			} else {
				handler, e = listener.createHttp2Portal(nk, router)
			}
			if e != nil {
				listener.Close()
				return
			}
			mux.Post(router.Pattern, handler)
		case http.MethodPut:
			if router.Portal.Tag == `` {
				handler, e = listener.createHttp2(dialers, router)
			} else {
				handler, e = listener.createHttp2Portal(nk, router)
			}
			if e != nil {
				listener.Close()
				return
			}
			mux.Put(router.Pattern, handler)
		case http.MethodPatch:
			if router.Portal.Tag == `` {
				handler, e = listener.createHttp2(dialers, router)
			} else {
				handler, e = listener.createHttp2Portal(nk, router)
			}
			if e != nil {
				listener.Close()
				return
			}
			mux.Patch(router.Pattern, handler)
		case `WS`:
			if router.Portal.Tag == `` {
				handler, e = listener.createWebsocket(dialers, router)
			} else {
				handler, e = listener.createWebsocketPortal(nk, router)
			}
			if e != nil {
				listener.Close()
				return
			}
			mux.Get(router.Pattern, handler)
		case `API`:
			for _, item := range api {
				pattern := path.Join(router.Pattern, item.Path)
				for _, method := range item.Method {
					switch method {
					case http.MethodGet:
						mux.Get(pattern, item.Handler)
						mux.Head(pattern, item.Handler)
						log.Info(`new api router`,
							`method`, method,
							`pattern`, pattern,
						)
					case http.MethodPost:
						mux.Post(pattern, item.Handler)
						log.Info(`new api router`,
							`method`, method,
							`pattern`, pattern,
						)
					case http.MethodPut:
						mux.Put(pattern, item.Handler)
						log.Info(`new api router`,
							`method`, method,
							`pattern`, pattern,
						)
					case http.MethodPatch:
						mux.Patch(pattern, item.Handler)
						log.Info(`new api router`,
							`method`, method,
							`pattern`, pattern,
						)
					case http.MethodDelete:
						mux.Delete(pattern, item.Handler)
						log.Info(`new api router`,
							`method`, method,
							`pattern`, pattern,
						)
					}
				}
			}
		default:
			listener.Close()
			e = errHttpMethod
			log.Warn(`http method not supported`,
				`error`, e,
				`method`, router.Method,
			)
			return
		}
	}
	listener.server.Handler = mux
	if !secure {
		var http2Server http2.Server
		listener.server.Handler = h2c.NewHandler(mux, &http2Server)
		e = http2.ConfigureServer(&listener.server, &http2Server)
		if e != nil {
			close(listener.done)
			l.Close()
			log.Error(`configure h2c server fail, used default close duration.`,
				`error`, e,
			)
			return
		}
	}
	return
}
func (l *HttpListener) access(r *http.Request, accessToken string) bool {
	if found, ok := r.Header[`Authorization`]; ok {
		for _, access := range found {
			if access == accessToken {
				return true
			}
		}
	}
	if found, ok := r.URL.Query()[`access_token`]; ok {
		for _, access := range found {
			if access == accessToken {
				return true
			}
		}
	}
	return false
}
func (l *HttpListener) createHttp2(dialers map[string]dialer.Dialer, router *config.Router) (handler http.HandlerFunc, e error) {
	log := l.log
	dialer, ok := dialers[router.Dialer.Tag]
	if !ok {
		e = errors.New(`dialer not found: ` + router.Dialer.Tag)
		log.Error(`dialer not found`, `dialer`, router.Dialer.Tag)
		return
	}
	var closeDuration time.Duration
	if router.Dialer.Close == `` {
		closeDuration = time.Second
	} else {
		var err error
		closeDuration, err = time.ParseDuration(router.Dialer.Close)
		if err != nil {
			closeDuration = time.Second
			log.Warn(`parse duration fail, used default close duration.`,
				`error`, err,
				`close`, router.Dialer.Close,
				`default`, closeDuration,
			)
		}
	}

	log = log.With(`method`, router.Method, `dialer`, router.Dialer.Tag)
	var accessToken string
	if router.Access != `` {
		accessToken = `Bearer ` + base64.RawURLEncoding.EncodeToString([]byte(router.Access))
	}
	if router.Access == `` {
		log.Info(`new router`,
			`pattern`, router.Pattern,
			`close`, closeDuration,
		)
	} else {
		log.Info(`new router`,
			`pattern`, router.Pattern,
			`access`, router.Access,
			`close`, closeDuration,
		)
	}
	handler = func(w http.ResponseWriter, r *http.Request) {
		if accessToken != `` && !l.access(r, accessToken) {
			log.Warn(`access not matched`)
			w.Header().Set(`Content-Type`, `text/plain; charset=utf-8`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`access not matched`))
			return
		}
		dst, e := dialer.Connect(r.Context())
		if e != nil {
			log.Warn(`connect fail`,
				`error`, e,
			)
			w.Header().Set(`Content-Type`, `text/plain; charset=utf-8`)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(e.Error()))
			return
		}
		addr := dst.RemoteAddr()
		log.Info(`bridge`,
			`network`, addr.Network,
			`addr`, addr.Addr,
			`secure`, addr.Secure,
			`url`, addr.URL,
		)
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		network.Bridging(ioutil.NewReadWriter(r.Body, w, r.Body), dst.ReadWriteCloser, l.pool, closeDuration)
	}
	return
}
func (l *HttpListener) createHttp2Portal(nk *network.Network, router *config.Router) (handler http.HandlerFunc, e error) {
	log := l.log.With(`portal`, router.Portal.Tag)
	var accessToken string
	if router.Access != `` {
		accessToken = `Bearer ` + base64.RawURLEncoding.EncodeToString([]byte(router.Access))
	}
	log = log.With(`method`, router.Method)
	listener := newHttpListener(l.done,
		network.NewAddr(`portal`, router.Portal.Tag),
	)
	dialer, e := nk.NewPortal(log, listener, &router.Portal)
	if e != nil {
		log.Error(`new portal listener fail`, `error`, e)
		return
	}
	go dialer.Serve()
	l.closer = append(l.closer, dialer)
	if router.Access == `` {
		log.Info(`new portal router`,
			`pattern`, router.Pattern,
		)
	} else {
		log.Info(`new portal router`,
			`pattern`, router.Pattern,
			`access`, router.Access,
		)
	}
	handler = func(w http.ResponseWriter, r *http.Request) {
		if accessToken != `` && !l.access(r, accessToken) {
			log.Warn(`access not matched`)
			w.Header().Set(`Content-Type`, `text/plain; charset=utf-8`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`access not matched`))
			return
		}
		w.WriteHeader(http.StatusOK)
		f, ok := w.(http.Flusher)
		if ok {
			f.Flush()
		}
		wc := newHttp2PortalWriter(w, f, r.Body)
		conn := ioutil.NewReadWriter(r.Body, wc, wc)
		select {
		case <-l.done:
			conn.Close()
			return
		case <-listener.selfDone:
			conn.Close()
			return
		case <-r.Context().Done():
			conn.Close()
			return
		case listener.ch <- conn:
			wc.Wait()
		}
	}
	return
}

type http2PortalWriter struct {
	w      io.Writer
	closer io.Closer
	e      error
	sync.Mutex
	f    http.Flusher
	done chan struct{}
}

func newHttp2PortalWriter(w io.Writer, f http.Flusher, closer io.Closer) *http2PortalWriter {
	return &http2PortalWriter{
		w:      w,
		f:      f,
		closer: closer,
		done:   make(chan struct{}),
	}
}
func (w *http2PortalWriter) Wait() {
	<-w.done
}
func (w *http2PortalWriter) Flush() {
	if w.f != nil {
		w.Lock()
		defer w.Unlock()
		w.f.Flush()
	}
}
func (w *http2PortalWriter) Close() (e error) {
	w.Lock()
	defer w.Unlock()
	if w.e != nil {
		e = w.e
		return
	}
	w.e = ErrClosed
	w.closer.Close()
	close(w.done)
	return
}
func (w *http2PortalWriter) Write(b []byte) (n int, e error) {
	w.Lock()
	defer w.Unlock()
	if w.e != nil {
		e = w.e
		return
	}
	n = len(b)
	if n == 0 {
		return
	}
	n, e = w.w.Write(b)
	if n != 0 && w.f != nil {
		w.f.Flush()
	}
	if e != nil {
		w.closer.Close()
		w.e = e
		close(w.done)
	}
	return
}

func (l *HttpListener) createWebsocketPortal(nk *network.Network, router *config.Router) (handler http.HandlerFunc, e error) {
	log := l.log.With(`portal`, router.Portal.Tag)
	var accessToken string
	if router.Access != `` {
		accessToken = `Bearer ` + base64.RawURLEncoding.EncodeToString([]byte(router.Access))
	}
	listener := newHttpListener(l.done,
		network.NewAddr(`portal`, router.Portal.Tag),
	)
	dialer, e := nk.NewPortal(log, listener, &router.Portal)
	if e != nil {
		log.Error(`new portal listener fail`, `error`, e)
		return
	}
	go dialer.Serve()
	l.closer = append(l.closer, dialer)
	if router.Access == `` {
		log.Info(`new portal router`,
			`pattern`, router.Pattern,
		)
	} else {
		log.Info(`new portal router`,
			`method`, `WebSocket`,
			`pattern`, router.Pattern,
			`access`, router.Access,
		)
	}
	upgrader := l.getUpgrader()
	handler = func(w http.ResponseWriter, r *http.Request) {
		if accessToken != `` && !l.access(r, accessToken) {
			log.Warn(`access not matched`)
			w.Header().Set(`Content-Type`, `text/plain; charset=utf-8`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`access not matched`))
			return
		}
		ws, e := upgrader.Upgrade(w, r, nil)
		if e != nil {
			log.Warn(`upgrade websocket fail`,
				`error`, e,
			)
			return
		}
		select {
		case <-l.done:
			ws.Close()
		case <-listener.selfDone:
			ws.Close()
		case <-r.Context().Done():
			ws.Close()
		case listener.ch <- httpmux.NewWebsocketConn(ws):
		}
	}
	return
}

func (l *HttpListener) createWebsocket(dialers map[string]dialer.Dialer, router *config.Router) (handler http.HandlerFunc, e error) {
	log := l.log
	dialer, ok := dialers[router.Dialer.Tag]
	if !ok {
		e = errors.New(`dialer not found: ` + router.Dialer.Tag)
		log.Error(`dialer not found`, `dialer`, router.Dialer.Tag)
		return
	}
	log = log.With(`method`, `WebSocket`, `dialer`, router.Dialer.Tag)
	var accessToken string
	if router.Access != `` {
		accessToken = `Bearer ` + base64.RawURLEncoding.EncodeToString([]byte(router.Access))
	}
	var closeDuration time.Duration
	if router.Dialer.Close == `` {
		closeDuration = time.Second
	} else {
		var err error
		closeDuration, err = time.ParseDuration(router.Dialer.Close)
		if err != nil {
			closeDuration = time.Second
			log.Warn(`parse duration fail, used default close duration.`,
				`error`, err,
				`close`, router.Dialer.Close,
				`default`, closeDuration,
			)
		}
	}
	if router.Access == `` {
		log.Info(`new router`,
			`pattern`, router.Pattern,
			`close`, closeDuration,
		)
	} else {
		log.Info(`new router`,
			`method`, `WebSocket`,
			`pattern`, router.Pattern,
			`access`, router.Access,
			`close`, closeDuration,
		)
	}
	upgrader := l.getUpgrader()
	handler = func(w http.ResponseWriter, r *http.Request) {
		if accessToken != `` && !l.access(r, accessToken) {
			log.Warn(`access not matched`)
			w.Header().Set(`Content-Type`, `text/plain; charset=utf-8`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`access not matched`))
			return
		}
		ws, e := upgrader.Upgrade(w, r, nil)
		if e != nil {
			log.Warn(`upgrade websocket fail`,
				`error`, e,
			)
			return
		}
		dst, e := dialer.Connect(r.Context())
		if e != nil {
			log.Warn(`connect fail`,
				`error`, e,
			)
			ws.Close()
			return
		}
		addr := dst.RemoteAddr()
		log.Info(`bridge`,
			`network`, addr.Network,
			`addr`, addr.Addr,
			`secure`, addr.Secure,
			`url`, addr.URL,
		)
		network.Bridging(httpmux.NewWebsocketConn(ws), dst.ReadWriteCloser, l.pool, closeDuration)
	}
	return
}
func (l *HttpListener) getUpgrader() *websocket.Upgrader {
	upgrader := l.upgrader
	if upgrader == nil {
		upgrader = &websocket.Upgrader{
			HandshakeTimeout: time.Second * 5,
			ReadBufferSize:   l.pool.Size(),
			WriteBufferSize:  l.pool.Size(),
			WriteBufferPool:  websocket.NewBufferPool(l.pool),
			CheckOrigin:      func(r *http.Request) bool { return true },
		}
		l.upgrader = upgrader
	}
	return upgrader
}
func (l *HttpListener) Close() (e error) {
	if l.closed == 0 && atomic.CompareAndSwapUint32(&l.closed, 0, 1) {
		close(l.done)
		for _, closer := range l.closer {
			closer.Close()
		}
		e = l.listener.Close()
	} else {
		e = ErrClosed
	}
	return
}
func (l *HttpListener) Serve() (e error) {
	if l.certFile != `` && l.keyFile != `` {
		e = l.server.ServeTLS(l.listener, l.certFile, l.keyFile)
	} else {
		e = l.server.Serve(l.listener)
	}
	return
}
