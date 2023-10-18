package agent

import (
	"bytes"
	"sync"
)

var (
	BufferSize = 1024 * 1024
	BufferPool = sync.Pool{New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, BufferSize))
	}}
)

type LogHolder struct {
	buffer *bytes.Buffer
	lock   sync.Mutex
}

func NewLogHolder() *LogHolder {
	return &LogHolder{
		buffer: BufferPool.Get().(*bytes.Buffer),
	}
}

func (b *LogHolder) Write(d []byte) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.buffer.Write(d)
}

func (b *LogHolder) GetAndClear() (*bytes.Buffer, bool) {
	b.lock.Lock()
	defer b.lock.Unlock()
	if b.buffer.Len() == 0 {
		return nil, false
	}
	resp := b.buffer
	b.buffer = BufferPool.Get().(*bytes.Buffer)
	return resp, true
}
