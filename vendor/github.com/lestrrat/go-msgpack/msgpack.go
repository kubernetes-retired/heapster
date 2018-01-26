//go:generate stringer -type Code
//go:generate go run internal/cmd/gencontainer/gencontainer.go - encoder_container_gen.go
//go:generate go run internal/cmd/gendecoder-numeric/gendecoder-numeric.go - decoder_numeric.go
//go:generate go run internal/cmd/genencoder-numeric/genencoder-numeric.go - encoder_numeric.go

package msgpack

import (
	"bytes"
	"sync"

	"github.com/pkg/errors"
)

var pool = sync.Pool{
	New: allocAppendingWriter,
}

func allocAppendingWriter() interface{} {
	return newAppendingWriter(9)
}

func releaseAppendingWriter(w *appendingWriter) {
	w.buf = w.buf[0:0]
	pool.Put(w)
}

// Marshal takes a Go value and serializes it in msgpack format.
func Marshal(v interface{}) ([]byte, error) {
	var buf = pool.Get().(*appendingWriter) // newAppendingWriter(9)
	defer releaseAppendingWriter(buf)
	if err := NewEncoder(buf).Encode(v); err != nil {
		return nil, errors.Wrap(err, `failed to marshal`)
	}
	return buf.Bytes(), nil
}

// Unmarshal takes a byte slice and a pointer to a Go value and
// deserializes the Go value from the data in msgpack format.
func Unmarshal(data []byte, v interface{}) error {
	buf := bytes.NewBuffer(data)
	if err := NewDecoder(buf).Decode(v); err != nil {
		return errors.Wrap(err, `failed to unmarshal`)
	}
	return nil
}
