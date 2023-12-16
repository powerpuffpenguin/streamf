package listener

import (
	"log/slog"

	"github.com/powerpuffpenguin/sf/config"
	"github.com/powerpuffpenguin/sf/dialer"
	"github.com/powerpuffpenguin/sf/pool"
)

type HttpListener struct {
}

func NewHttpListener(log *slog.Logger, pool *pool.Pool, dialers map[string]dialer.Dialer, opts *config.BasicListener) (listener *HttpListener, e error) {
	return
}
func (l *HttpListener) Close() (e error) {
	return
}
func (l *HttpListener) Serve() (e error) {
	return
}
