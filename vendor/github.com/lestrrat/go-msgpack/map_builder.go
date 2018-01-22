package msgpack

import (
	"bytes"
	"io"
	"math"

	"github.com/pkg/errors"
)

type mapBuilder struct {
	buffer []interface{}
}

func NewMapBuilder() MapBuilder {
	return &mapBuilder{}
}

func (b *mapBuilder) Reset() {
	b.buffer = b.buffer[:0]
}

func (b *mapBuilder) Add(key string, value interface{}) {
	b.buffer = append(b.buffer, key, value)
}

func (b *mapBuilder) Count() int {
	return len(b.buffer) / 2
}

func WriteMapHeader(dst io.Writer, c int) error {
	var w Writer
	var ok bool
	if w, ok = dst.(Writer); !ok {
		w = NewWriter(dst)
	}

	switch {
	case c < 16:
		w.WriteByte(FixMap0.Byte() + byte(c))
	case c < math.MaxUint16:
		w.WriteByte(Map16.Byte())
		w.WriteUint16(uint16(c))
	case c < math.MaxUint32:
		w.WriteByte(Map32.Byte())
		w.WriteUint32(uint32(c))
	default:
		return errors.Errorf(`map builder: map element count out of range (%d)`, c)
	}
	return nil
}

func (b *mapBuilder) Encode(dst io.Writer) error {
	WriteMapHeader(dst, b.Count())

	e := NewEncoder(dst)
	for i := 0; i < b.Count(); i++ {
		if err := e.Encode(b.buffer[i*2]); err != nil {
			return errors.Wrapf(err, `map builder: failed to encode map key %s`, b.buffer[i])
		}
		if err := e.Encode(b.buffer[i*2+1]); err != nil {
			return errors.Wrapf(err, `map builder: failed to encode map element for %s`, b.buffer[i])
		}
	}
	return nil
}

func (b *mapBuilder) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := b.Encode(&buf); err != nil {
		return nil, errors.Wrap(err, `map builder: failed to write map`)
	}

	return buf.Bytes(), nil
}
