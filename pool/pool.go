package pool

import (
	"sync"

	"github.com/powerpuffpenguin/sf/config"
)

type Pool struct {
	ch   chan []byte
	size int
	pool sync.Pool
}

func New(conf *config.Pool) *Pool {
	size := conf.Size
	cache := conf.Cache
	if size < 1024*4 {
		size = 1024 * 32
	}
	var ch chan []byte
	if cache > 0 {
		ch = make(chan []byte, cache)
	}
	p := &Pool{
		ch:   ch,
		size: size,
	}
	p.pool.New = func() any {
		return make([]byte, size)
	}
	return p
}
func (p *Pool) Size() int {
	return p.size
}
func (p *Pool) Get() []byte {
	if p.ch != nil {
		select {
		case b := <-p.ch:
			return b[:p.size]
		default:
		}
	}
	return p.pool.Get().([]byte)[:p.size]
}
func (p *Pool) Put(b []byte) {
	if cap(b) < p.size {
		return
	} else if p.ch != nil {
		select {
		case p.ch <- b:
			return
		default:
		}
	}

	p.pool.Put(b)
}
