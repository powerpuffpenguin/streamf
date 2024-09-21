package udp

import (
	"log/slog"
	"net"
	"time"

	"github.com/powerpuffpenguin/streamf/config"
)

type UDP struct {
	c       *net.UDPConn
	to      *net.UDPAddr
	timeout time.Duration
	size    int
}

func New(log *slog.Logger, opts *config.UDPForward) (u *UDP, e error) {
	tag := opts.Tag
	if tag == `` {
		tag = `udp ` + opts.Listen + ` -> ` + opts.To
	}
	log = log.With(
		`tag`, tag,
		`listener`, opts.Listen,
		`to`, opts.To)
	addr, e := net.ResolveUDPAddr(`udp`, opts.Listen)
	if e != nil {
		log.Error(`listen udp fial`, `error`, e)
		return
	}
	c, e := net.ListenUDP(`udp`, addr)
	if e != nil {
		log.Error(`listen udp fial`, `error`, e)
		return
	}
	to, e := net.ResolveUDPAddr(`udp`, opts.To)
	if e != nil {
		log.Error(`listen udp fial`, `error`, e)
		return
	}
	var timeout time.Duration
	if opts.Timeout == `` {
		timeout = time.Minute * 3
	} else {
		var err error
		timeout, err = time.ParseDuration(opts.Timeout)
		if err != nil {
			timeout = time.Minute * 3
		}
	}
	size := opts.Size
	if size < 128 {
		size = 1024 * 2
	}
	log.Info(`udp forward`, `timeout`, timeout, `size`, size)
	u = &UDP{
		c:       c,
		to:      to,
		timeout: timeout,
		size:    size,
	}
	return
}
func (u *UDP) Serve() {

}
func (u *UDP) Close() {

}
