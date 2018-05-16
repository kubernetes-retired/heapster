package msgpack

import (
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// NewEncoder creates a new Encoder that writes serialized forms
// to the specified io.Writer
//
// Note that Encoders are NEVER meant to be shared concurrently
// between goroutines. You DO NOT write serialized data concurrently
// to the same destination.
func NewEncoder(w io.Writer) *Encoder {
	var dst Writer
	if x, ok := w.(Writer); ok {
		dst = x
	} else {
		dst = NewWriter(w)
	}

	return &Encoder{
		dst: dst,
	}
}

func inPositiveFixNumRange(i int64) bool {
	return i >= 0 && i <= 127
}

func inNegativeFixNumRange(i int64) bool {
	return i >= -31 && i <= -1
}

func isExtType(t reflect.Type) (int, bool) {
	muExtEncode.RLock()
	typ, ok := extEncodeRegistry[t]
	muExtEncode.RUnlock()
	if ok {
		return typ, true
	}

	return 0, false
}

func isEncodeMsgpacker(t reflect.Type) bool {
	return t.Implements(encodeMsgpackerType)
}

func (e *Encoder) Writer() Writer {
	return e.dst
}

func (e *Encoder) Encode(v interface{}) error {
	switch v := v.(type) {
	case string:
		return e.EncodeString(v)
	case []byte:
		return e.EncodeBytes(v)
	case bool:
		return e.EncodeBool(v)
	case float32:
		return e.EncodeFloat32(v)
	case float64:
		return e.EncodeFloat64(v)
	case uint:
		return e.EncodeUint64(uint64(v))
	case uint8:
		return e.EncodeUint8(v)
	case uint16:
		return e.EncodeUint16(v)
	case uint32:
		return e.EncodeUint32(v)
	case uint64:
		return e.EncodeUint64(v)
	case int:
		return e.EncodeInt64(int64(v))
	case int8:
		return e.EncodeInt8(v)
	case int16:
		return e.EncodeInt16(v)
	case int32:
		return e.EncodeInt32(v)
	case int64:
		return e.EncodeInt64(v)
	}

	// Find the first non-pointer, non-interface{}
	rv := reflect.ValueOf(v)
INDIRECT:
	for {
		if !rv.IsValid() {
			return e.EncodeNil()
		}

		if _, ok := isExtType(rv.Type()); ok {
			return e.EncodeExt(rv.Interface().(EncodeMsgpacker))
		}

		if ok := isEncodeMsgpacker(rv.Type()); ok {
			return rv.Interface().(EncodeMsgpacker).EncodeMsgpack(e)
		}
		switch rv.Kind() {
		case reflect.Ptr, reflect.Interface:
			rv = rv.Elem()
		default:
			break INDIRECT
		}
	}

	if !rv.IsValid() {
		return e.EncodeNil()
	}

	v = rv.Interface()
	switch rv.Kind() {
	case reflect.Slice:
		return e.EncodeArray(v)
	case reflect.Map:
		return e.EncodeMap(v)
	case reflect.Struct:
		return e.EncodeStruct(v)
	}

	return errors.Errorf(`msgpack: encode unimplemented for type %s`, rv.Type())
}

func (e *Encoder) encodePositiveFixNum(i uint8) error {
	return e.dst.WriteByte(byte(i))
}

func (e *Encoder) encodeNegativeFixNum(i int8) error {
	return e.dst.WriteByte(byte(i))
}

func (e *Encoder) EncodeNil() error {
	return e.dst.WriteByte(Nil.Byte())
}

func (e *Encoder) EncodeBool(b bool) error {
	var code Code
	if b {
		code = True
	} else {
		code = False
	}
	return e.dst.WriteByte(code.Byte())
}

func (e *Encoder) EncodePositiveFixNum(i uint8) error {
	panic(fmt.Sprintf("fuck fixnum i = %d, max = %d", i, uint8(MaxPositiveFixNum)))

	if i > uint8(MaxPositiveFixNum) || i < 0 {
		return errors.Errorf(`msgpack: value %d is not in range for positive FixNum (127 >= x >= 0)`, i)
	}

	if err := e.dst.WriteByte(byte(i)); err != nil {
		return errors.Wrap(err, `msgpack: failed to write FixNum`)
	}
	return nil
}

func (e *Encoder) EncodeNegativeFixNum(i int8) error {
	if i < -31 || i >= 0 {
		return errors.Errorf(`msgpack: value %d is not in range for positive FixNum (0 > x >= -31)`, i)
	}

	if err := e.dst.WriteByte(byte(i)); err != nil {
		return errors.Wrap(err, `msgpack: failed to write FixNum`)
	}
	return nil
}

func (e *Encoder) EncodeBytes(b []byte) error {
	l := len(b)

	var w int
	var code Code
	switch {
	case l <= math.MaxUint8:
		code = Bin8
		w = 1
	case l <= math.MaxUint16:
		code = Bin16
		w = 2
	case l <= math.MaxUint32:
		code = Bin32
		w = 4
	default:
		return errors.Errorf(`msgpack: string is too long (len=%d)`, l)
	}

	if err := e.writePreamble(code, w, l); err != nil {
		return errors.Wrap(err, `msgpack: failed to write []byte preamble`)
	}
	e.dst.Write(b)
	return nil
}

func (e *Encoder) EncodeString(s string) error {
	l := len(s)
	switch {
	case l < 32:
		e.dst.WriteByte(FixStr0.Byte() | uint8(l))
	case l <= math.MaxUint8:
		e.dst.WriteByte(Str8.Byte())
		e.dst.WriteUint8(uint8(l))
	case l <= math.MaxUint16:
		e.dst.WriteByte(Str16.Byte())
		e.dst.WriteUint16(uint16(l))
	case l <= math.MaxUint32:
		e.dst.WriteByte(Str32.Byte())
		e.dst.WriteUint32(uint32(l))
	default:
		return errors.Errorf(`msgpack: string is too long (len=%d)`, l)
	}

	e.dst.WriteString(s)
	return nil
}

func (e *Encoder) writePreamble(code Code, w int, l int) error {
	if err := e.dst.WriteByte(code.Byte()); err != nil {
		return errors.Wrap(err, `msgpack: failed to write code`)
	}

	switch w {
	case 1:
		if err := e.dst.WriteUint8(uint8(l)); err != nil {
			return errors.Wrap(err, `msgpack: failed to write length`)
		}
	case 2:
		if err := e.dst.WriteUint16(uint16(l)); err != nil {
			return errors.Wrap(err, `msgpack: failed to write length`)
		}
	case 4:
		if err := e.dst.WriteUint32(uint32(l)); err != nil {
			return errors.Wrap(err, `msgpack: failed to write length`)
		}
	}
	return nil
}

func (e *Encoder) EncodeArrayHeader(l int) error {
	if err := WriteArrayHeader(e.dst, l); err != nil {
		return errors.Wrap(err, `msgpack: failed to write array header`)
	}
	return nil
}

func (e *Encoder) EncodeArray(v interface{}) error {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
	default:
		return errors.Errorf(`msgpack: argument must be an array or a slice`)
	}

	if err := e.EncodeArrayHeader(rv.Len()); err != nil {
		return err
	}

	switch rv.Type().Elem().Kind() {
	case reflect.String:
		return e.encodeArrayString(v)
	case reflect.Bool:
		return e.encodeArrayBool(v)
	case reflect.Int:
		return e.encodeArrayInt(v)
	case reflect.Int8:
		return e.encodeArrayInt8(v)
	case reflect.Int16:
		return e.encodeArrayInt16(v)
	case reflect.Int32:
		return e.encodeArrayInt32(v)
	case reflect.Int64:
		return e.encodeArrayInt64(v)
	case reflect.Uint:
		return e.encodeArrayUint(v)
	case reflect.Uint8:
		return e.encodeArrayUint8(v)
	case reflect.Uint16:
		return e.encodeArrayUint16(v)
	case reflect.Uint32:
		return e.encodeArrayUint32(v)
	case reflect.Uint64:
		return e.encodeArrayUint64(v)
	case reflect.Float32:
		return e.encodeArrayFloat32(v)
	case reflect.Float64:
		return e.encodeArrayFloat64(v)
	}

	for i := 0; i < rv.Len(); i++ {
		if err := e.Encode(rv.Index(i).Interface()); err != nil {
			return errors.Wrap(err, `msgpack: failed to write array payload`)
		}
	}
	return nil
}

func (e *Encoder) EncodeMap(v interface{}) error {
	rv := reflect.ValueOf(v)

	if !rv.IsValid() {
		return e.EncodeNil()
	}

	if rv.Kind() != reflect.Map {
		var typ string
		if !rv.IsValid() {
			typ = "invalid"
		} else {
			typ = rv.Type().String()
		}
		return errors.Errorf(`msgpack: argument to EncodeMap must be a map (not %s)`, typ)
	}

	if rv.IsNil() {
		return e.EncodeNil()
	}

	if rv.Type().Key().Kind() != reflect.String {
		return errors.Errorf(`msgpack: keys to maps must be strings (not %s)`, rv.Type().Key())
	}

	// XXX We do NOT use MapBuilder's convenience methods except for the
	// WriteHeader bit, purely for performance reasons.
	keys := rv.MapKeys()
	WriteMapHeader(e.dst, len(keys))

	// These are silly fast paths for common cases
	switch rv.Type().Elem().Kind() {
	case reflect.String:
		return e.encodeMapString(v)
	case reflect.Bool:
		return e.encodeMapBool(v)
	case reflect.Uint:
		return e.encodeMapUint(v)
	case reflect.Uint8:
		return e.encodeMapUint8(v)
	case reflect.Uint16:
		return e.encodeMapUint16(v)
	case reflect.Uint32:
		return e.encodeMapUint32(v)
	case reflect.Uint64:
		return e.encodeMapUint64(v)
	case reflect.Int:
		return e.encodeMapInt(v)
	case reflect.Int8:
		return e.encodeMapInt8(v)
	case reflect.Int16:
		return e.encodeMapInt16(v)
	case reflect.Int32:
		return e.encodeMapInt32(v)
	case reflect.Int64:
		return e.encodeMapInt64(v)
	case reflect.Float32:
		return e.encodeMapFloat32(v)
	case reflect.Float64:
		return e.encodeMapFloat64(v)
	default:
		for _, key := range keys {
			if err := e.EncodeString(key.Interface().(string)); err != nil {
				return errors.Wrap(err, `failed to encode map key`)
			}

			if err := e.Encode(rv.MapIndex(key).Interface()); err != nil {
				return errors.Wrap(err, `failed to encode map value`)
			}
		}
	}
	return nil
}

var tags = []string{`msgpack`, `msg`}

func parseMsgpackTag(rv reflect.StructField) (string, bool) {
	var name = rv.Name
	var omitempty bool

	// We will support both msg and msgpack tags, the former
	// is used by tinylib/msgp, and the latter vmihailenco/msgpack
LOOP:
	for _, tagName := range tags {
		if tag, ok := rv.Tag.Lookup(tagName); ok && tag != "" {
			l := strings.Split(tag, ",")
			if len(l) > 0 && l[0] != "" {
				name = l[0]
			}

			if len(l) > 1 && l[1] == "omitempty" {
				omitempty = true
			}
			break LOOP
		}
	}
	return name, omitempty
}

// EncodeStruct encodes a struct value as a map object.
func (e *Encoder) EncodeStruct(v interface{}) error {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return e.EncodeNil()
	}

	if _, ok := isExtType(rv.Type()); ok {
		return e.EncodeExt(v.(EncodeMsgpacker))
	}

	if v, ok := v.(EncodeMsgpacker); ok {
		return v.EncodeMsgpack(e)
	}

	if rv.Kind() != reflect.Struct {
		return errors.Errorf(`msgpack: argument to EncodeStruct must be a struct (not %s)`, rv.Type())
	}
	mapb := NewMapBuilder()

	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		ft := rt.Field(i)
		if ft.PkgPath != "" {
			continue
		}

		name, omitempty := parseMsgpackTag(ft)
		if name == "-" {
			continue
		}

		field := rv.Field(i)
		if omitempty {
			if reflect.DeepEqual(field.Interface(), reflect.Zero(field.Type()).Interface()) {
				continue
			}
		}

		mapb.Add(name, field.Interface())
	}

	if err := mapb.Encode(e.dst); err != nil {
		return errors.Wrap(err, `msgpack: failed to write map payload`)
	}
	return nil
}

func (e *Encoder) EncodeExtType(v EncodeMsgpacker) error {
	t := reflect.TypeOf(v)

	muExtDecode.RLock()
	typ, ok := extEncodeRegistry[t]
	muExtDecode.RUnlock()

	if !ok {
		return errors.Errorf(`msgpack: type %s has not been registered as an extension`, reflect.TypeOf(v))
	}

	if err := e.dst.WriteByte(byte(typ)); err != nil {
		return errors.Wrapf(err, `msgpack: failed to write ext type for %s`, t)
	}
	return nil
}

func (e *Encoder) EncodeExt(v EncodeMsgpacker) error {
	w := newAppendingWriter(9)
	elocal := NewEncoder(w)

	if err := v.EncodeMsgpack(elocal); err != nil {
		return errors.Wrapf(err, `msgpack: failed during call to EncodeMsgpack for %s`, reflect.TypeOf(v))
	}

	buf := w.Bytes()
	e.EncodeExtHeader(len(buf))
	e.EncodeExtType(v)
	for b := buf; len(b) > 0; {
		n, err := e.dst.Write(buf)
		b = b[n:]
		if err != nil {
			return errors.Wrap(err, `msgpack: failed to write extension payload`)
		}
	}

	return nil
}

func (e *Encoder) EncodeExtHeader(l int) error {
	switch {
	case l == 1:
		if err := e.dst.WriteByte(FixExt1.Byte()); err != nil {
			return errors.Wrap(err, `msgpack: failed to write fixext1 code`)
		}
	case l == 2:
		if err := e.dst.WriteByte(FixExt2.Byte()); err != nil {
			return errors.Wrap(err, `msgpack: failed to write fixext2 code`)
		}
	case l == 4:
		if err := e.dst.WriteByte(FixExt4.Byte()); err != nil {
			return errors.Wrap(err, `msgpack: failed to write fixext4 code`)
		}
	case l == 8:
		if err := e.dst.WriteByte(FixExt8.Byte()); err != nil {
			return errors.Wrap(err, `msgpack: failed to write fixext8 code`)
		}
	case l == 16:
		if err := e.dst.WriteByte(FixExt16.Byte()); err != nil {
			return errors.Wrap(err, `msgpack: failed to write fixext16 code`)
		}
	case l <= math.MaxUint8:
		if err := e.dst.WriteByteUint8(Ext8.Byte(), uint8(l)); err != nil {
			return errors.Wrap(err, `msgpack: failed to write ext8 code and payload length`)
		}
	case l <= math.MaxUint16:
		if err := e.dst.WriteByteUint16(Ext16.Byte(), uint16(l)); err != nil {
			return errors.Wrap(err, `msgpack: failed to write ext16 code and payload length`)
		}
	case l <= math.MaxUint32:
		if err := e.dst.WriteByteUint32(Ext32.Byte(), uint32(l)); err != nil {
			return errors.Wrap(err, `msgpack: failed to write ext32 code and payload length`)
		}
	default:
		return errors.Errorf(`msgpack: extension payload too large: %d bytes`, l)
	}

	return nil
}
