package bufferpool

import (
	"bytes"
	"sync"
)

// Package bufferpool is a simple wrapper around sync.Pool that is specific
// to bytes.Buffer.

var global = New()

func Get() *bytes.Buffer {
	return global.Get()
}

func Release(buf *bytes.Buffer) {
	global.Release(buf)
}

type BufferPool struct {
	pool sync.Pool
}

func New() *BufferPool {
	var bp BufferPool
	bp.pool.New = allocBuffer
	return &bp
}

func allocBuffer() interface{} {
	return &bytes.Buffer{}
}

func (bp *BufferPool) Get() *bytes.Buffer {
	return bp.pool.Get().(*bytes.Buffer)
}

func (bp *BufferPool) Release(buf *bytes.Buffer) {
	buf.Reset()
	bp.pool.Put(buf)
}
