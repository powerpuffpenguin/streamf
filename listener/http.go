package listener

import (
	"encoding/base64"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/dialer"
	"github.com/powerpuffpenguin/streamf/internal/httpmux"
	"github.com/powerpuffpenguin/streamf/internal/network"
	"github.com/powerpuffpenguin/streamf/ioutil"
	"github.com/powerpuffpenguin/streamf/pool"
	"github.com/powerpuffpenguin/streamf/third-party/websocket"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type HttpListener struct {
	server            http.Server
	listener          net.Listener
	certFile, keyFile string
	pool              *pool.Pool
	log               *slog.Logger
	closed            uint32
	upgrader          *websocket.Upgrader
}

func NewHttpListener(nk *network.Network, log *slog.Logger, pool *pool.Pool, dialers map[string]dialer.Dialer, opts *config.BasicListener, routers []*config.Router) (listener *HttpListener, e error) {
	var (
		l      net.Listener
		secure bool
	)
	if opts.TLS.CertFile != `` && opts.TLS.KeyFile != `` {
		secure = true
	}
	l, e = nk.Listen(opts.Network, opts.Address)
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

	log.Info(`new http listener`)
	listener = &HttpListener{
		certFile: opts.TLS.CertFile,
		keyFile:  opts.TLS.KeyFile,
		listener: l,
		pool:     pool,
		log:      log,
	}
	var (
		mux     = httpmux.New(log)
		handler http.HandlerFunc
	)
	for _, router := range routers {
		switch strings.ToUpper(router.Method) {
		case ``, http.MethodPost:
			handler, e = listener.createHttp2(dialers, router)
			if e != nil {
				l.Close()
				return
			}
			mux.Post(router.Pattern, handler)
		case http.MethodPut:
			handler, e = listener.createHttp2(dialers, router)
			if e != nil {
				l.Close()
				return
			}
			mux.Put(router.Pattern, handler)
		case http.MethodPatch:
			handler, e = listener.createHttp2(dialers, router)
			if e != nil {
				l.Close()
				return
			}
			mux.Patch(router.Pattern, handler)
		case `WS`:
			handler, e = listener.createWebsocket(dialers, router)
			if e != nil {
				l.Close()
				return
			}
			mux.Get(router.Pattern, handler)
		default:
			l.Close()
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
		bridging(ioutil.NewReadWriter(r.Body, w, r.Body), dst.ReadWriteCloser, l.pool, closeDuration)
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
		bridging(httpmux.NewWebsocketConn(ws), dst.ReadWriteCloser, l.pool, closeDuration)
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
