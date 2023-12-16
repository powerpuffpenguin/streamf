package websocket

type BytesPool interface {
	Get() []byte
	Put([]byte)
}
type bufferPool struct {
	bp BytesPool
}

func NewBufferPool(bp BytesPool) BufferPool {
	return bufferPool{
		bp: bp,
	}
}

// Get gets a value from the pool or returns nil if the pool is empty.
func (b bufferPool) Get() interface{} {
	return writePoolData{
		buf: b.bp.Get(),
	}
}

// Put adds a value to the pool.
func (b bufferPool) Put(a interface{}) {
	wpd, _ := a.(writePoolData)
	b.bp.Put(wpd.buf)
}
