package fluent

import (
	"time"

	msgpack "github.com/lestrrat/go-msgpack"
	"github.com/pkg/errors"
)

func init() {
	if err := msgpack.RegisterExt(0, EventTime{}); err != nil {
		panic(err)
	}
}

// DecodeMsgpack decodes from a msgpack stream and materializes
// a EventTime object
func (t *EventTime) DecodeMsgpack(d *msgpack.Decoder) error {
	r := d.Reader()

	sec, err := r.ReadUint32()
	if err != nil {
		return errors.Wrap(err, `failed to read uint32 from first 4 bytes`)
	}

	nsec, err := r.ReadUint32()
	if err != nil {
		return errors.Wrap(err, `failed to read uint32 from second 4 bytes`)
	}

	t.Time = time.Unix(int64(sec), int64(nsec)).UTC()
	return nil
}

// EncodeMsgpack encodes the EventTime into msgpack format
func (t EventTime) EncodeMsgpack(e *msgpack.Encoder) error {
	w := e.Writer()
	if err := w.WriteUint32(uint32(t.Unix())); err != nil {
		return errors.Wrap(err, `failed to write EventTime seconds payload`)
	}

	if err := w.WriteUint32(uint32(t.Nanosecond())); err != nil {
		return errors.Wrap(err, `failed to write EventTime nanoseconds payload`)
	}

	return nil
}
