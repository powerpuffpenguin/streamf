package listener

import (
	"log/slog"

	"github.com/powerpuffpenguin/sf/config"
	"github.com/powerpuffpenguin/sf/dialer"
)

type HttpListener struct {
}

func NewHttpListener(log *slog.Logger, dialers map[string]dialer.Dialer, opts *config.BasicListener) (listener *HttpListener, e error) {
	return
}
func (l *HttpListener) Close() (e error) {
	return
}
func (l *HttpListener) Serve() (e error) {
	return
}
