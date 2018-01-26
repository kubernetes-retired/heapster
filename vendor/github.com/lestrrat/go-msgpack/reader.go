package msgpack

import (
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
)

type reader struct {
	src io.Reader
	// Note: accessing buf concurrently is a mistake. But you DO NOT
	// write to a writer concurrently, or otherwise you can't guarantee
	// the correct memory layout. We assume that the caller doesn't do
	// anything silly.
	buf []byte
}

func NewReader(r io.Reader) Reader {
	return &reader{
		src: r,
		buf: make([]byte, 9),
	}
}

func (r *reader) Read(buf []byte) (int, error) {
	return r.src.Read(buf)
}

func (r *reader) ReadByte() (byte, error) {
	b := r.buf[:1]
	n, err := r.src.Read(b)
	if n != 1 {
		return byte(0), errors.Wrap(err, `reader: failed to read byte`)
	}
	return b[0], nil
}

func (r *reader) ReadUint8() (uint8, error) {
	b, err := r.ReadByte()
	if err != nil {
		return uint8(0), errors.Wrap(err, `reader: failed to read uint8`)
	}
	return uint8(b), nil
}

func (r *reader) ReadUint16() (uint16, error) {
	b := r.buf[:2]
	if _, err := r.src.Read(b); err != nil {
		return uint16(0), errors.Wrap(err, `reader: failed to read uint16`)
	}
	return uint16(b[1]) | uint16(b[0])<<8, nil
}

func (r *reader) ReadUint32() (uint32, error) {
	b := r.buf[:4]
	if _, err := r.src.Read(b); err != nil {
		return uint32(0), errors.Wrap(err, `reader: failed to read uint32`)
	}
	return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24, nil
}

func (r *reader) ReadUint64() (uint64, error) {
	b := r.buf[:8]
	if _, err := r.src.Read(b); err != nil {
		return uint64(0), errors.Wrap(err, `reader: failed to read uint64`)
	}
	return uint64(b[7]) | uint64(b[6])<<8 | uint64(b[5])<<16 | uint64(b[4])<<24 |
		uint64(b[3])<<32 | uint64(b[2])<<40 | uint64(b[1])<<48 | uint64(b[0])<<56, nil
}

func (r *reader) readbuf(size int) error {
	b := r.buf[:size]
	for len(b) > 0 {
		n, err := r.src.Read(b)
		b = b[n:]
		if err != nil {
			return errors.Wrapf(err, `reader: failed to read %d bytes`, size)
		}
	}
	return nil
}

func (r *reader) ReadByteUint8() (byte, uint8, error) {
	const size = 2
	if err := r.readbuf(size); err != nil {
		return 0, 0, err
	}
	return r.buf[0], uint8(r.buf[1]), nil
}

func (r *reader) ReadByteUint16() (byte, uint16, error) {
	const size = 3
	if err := r.readbuf(size); err != nil {
		return 0, 0, err
	}
	return r.buf[0], binary.BigEndian.Uint16(r.buf[1:]), nil
}

func (r *reader) ReadByteUint32() (byte, uint32, error) {
	const size = 5
	if err := r.readbuf(size); err != nil {
		return 0, 0, err
	}
	return r.buf[0], binary.BigEndian.Uint32(r.buf[1:]), nil
}

func (r *reader) ReadByteUint64() (byte, uint64, error) {
	const size = 9
	if err := r.readbuf(size); err != nil {
		return 0, 0, err
	}
	return r.buf[0], binary.BigEndian.Uint64(r.buf[1:]), nil
}
