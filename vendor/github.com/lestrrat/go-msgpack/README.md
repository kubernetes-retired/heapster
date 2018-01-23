# go-msgpack

A `msgpack` serializer and deserializer

[![Build Status](https://travis-ci.org/lestrrat/go-msgpack.png?branch=master)](https://travis-ci.org/lestrrat/go-msgpack)

[![GoDoc](https://godoc.org/github.com/lestrrat/go-msgpack?status.svg)](https://godoc.org/github.com/lestrrat/go-msgpack)

# SYNOPSIS

```go
package msgpack_test

import (
  "fmt"
  "time"

  msgpack "github.com/lestrrat/go-msgpack"
  "github.com/pkg/errors"
)

type EventTime struct {
  time.Time
}

func init() {
  if err := msgpack.RegisterExt(0, EventTime{}); err != nil {
    panic(err)
  }
}

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

type FluentdMessage struct {
  Tag    string
  Time   EventTime
  Record map[string]interface{}
  Option map[string]interface{}
}

func (m FluentdMessage) EncodeMsgpack(e *msgpack.Encoder) error {
  if err := e.EncodeArrayHeader(4); err != nil {
    return errors.Wrap(err, `failed to encode array header`)
  }
  if err := e.EncodeString(m.Tag); err != nil {
    return errors.Wrap(err, `failed to encode tag`)
  }
  if err := e.EncodeStruct(m.Time); err != nil {
    return errors.Wrap(err, `failed to encode time`)
  }
  if err := e.EncodeMap(m.Record); err != nil {
    return errors.Wrap(err, `failed to encode record`)
  }
  if err := e.EncodeMap(m.Option); err != nil {
    return errors.Wrap(err, `failed to encode option`)
  }
  return nil
}

func (m *FluentdMessage) DecodeMsgpack(e *msgpack.Decoder) error {
  var l int
  if err := e.DecodeArrayLength(&l); err != nil {
    return errors.Wrap(err, `failed to decode msgpack array length`)
  }

  if l != 4 {
    return errors.Errorf(`invalid array length %d (expected 4)`, l)
  }

  if err := e.DecodeString(&m.Tag); err != nil {
    return errors.Wrap(err, `failed to decode fluentd message tag`)
  }

  if err := e.DecodeStruct(&m.Time); err != nil {
    return errors.Wrap(err, `failed to decode fluentd time`)
  }

  if err := e.DecodeMap(&m.Record); err != nil {
    return errors.Wrap(err, `failed to decode fluentd record`)
  }

  if err := e.DecodeMap(&m.Option); err != nil {
    return errors.Wrap(err, `failed to decode fluentd option`)
  }

  return nil
}

func ExampleFluentdMessage() {
  var f1 = FluentdMessage{
    Tag:  "foo",
    Time: EventTime{Time: time.Unix(1234567890, 123).UTC()},
    Record: map[string]interface{}{
      "count": 1000,
    },
  }

  b, err := msgpack.Marshal(f1)
  if err != nil {
    fmt.Printf("%s\n", err)
    return
  }

  var f2 FluentdMessage
  if err := msgpack.Unmarshal(b, &f2); err != nil {
    fmt.Printf("%s\n", err)
    return
  }

  fmt.Printf("%s %s %v %v\n", f2.Tag, f2.Time, f2.Record, f2.Option)
  // OUTPUT:
  // foo 2009-02-13 23:31:30.000000123 +0000 UTC map[count:1000] map[]
}

func ExampleEventTime() {
  var e1 = EventTime{Time: time.Unix(1234567890, 123).UTC()}

  b, err := msgpack.Marshal(e1)
  if err != nil {
    fmt.Printf("%s\n", err)
    return
  }

  var e2 interface{}
  if err := msgpack.Unmarshal(b, &e2); err != nil {
    fmt.Printf("%s\n", err)
    return
  }
  // OUTPUT:
}
```

# STATUS

* Requires more testing for array/map/struct types

# DESCRIPTION

While tinkering with low-level `msgpack` stuff for the first time,
I realized that I didn't know enough about its internal workings to make
suggestions of have confidence producing bug reports, and I really
should: So I wrote one for my own amusement and education.

# FEATURES

## API Compatibility With stdlib

`github.com/vmihailenco/msgpack.v2`, which this library was initially
based upon, has subtle differences with the stdlib. For example,
`"github.com/vmihailenco/msgpack.v2".Decoder.Decode()` has a signature
of `Decode(v ...interface{})`, which doesn't match with the signature
in, for example, `encoding/json`. This subtle difference makes it hard
to use interfaces to make swappable serializers.

Also, all decoding API takes an argument to be assigned to instead of
returning a value.

## Custom Serialization

If you would like to customize serialization for a particular type,
you can create a type that implements the `msgpack.EncodeMsgpacker`
and/or `msgpack.DecodeMsgpacker` interface.

```go
func (v *Object) EncodeMsgpack(e *msgpack.Encoder) error {
  ...
}

func (v *Object) DecodeMsgpack(d *msgpack.Decoder) error {
  ...
}
```

## Low Level Writer/Reader

In some rare cases, such as when you are creating extensions, you need
a more fine grained control on what you read or write. For this, you may
use the `msgpack.Writer` and `msgpack.Reader` objects.

These objects know how to read or write bytes of data in the correct
byte order.

## Struct Tags

Struct tags are supported via the `msgpack` keyword. The syntax follows that of 
`encoding/json` package:

```go
type Example struct {
    Foo struct `msgpack:"foo"`
    Bar struct `msgpack:"bar,omitempty"`
}
```

For convenience for those migrating from github.com/tinylib/msgpack, we also
support the "msg" struct tag.

# PROS/CONS

## PROS

As most late comers are, I believe the project is a little bit cleaner than my predecessors, which **possibly** could mean a slightly easier experience for the users to hack and tweak it. I know, it's very subjective.

As far as comparisons against `gopkg.in/vmihailenco/msgpack.v2` goes, this library tries to keep the API as compatible as possible to the standard library's `encoding/*` packages. For example, `encoding/json` allows:

```go
  b, _ := json.Marshal(true)

  // using uninitialized empty interface
  var v interface{}
  json.Unmarshal(b, &v)
```

But if you do the same with `gopkg.in/vmihailenco/msgpack.v2`, this throws a panic:

```go
  b, _ := msgpack.Marshal(true)

  // using uninitialized empty interface
  var v interface{}
  msgpack.Unmarshal(b, &v)
```

This library follows the semantics for `encoding/json`, and you can safely pass an uninitialized empty inteface to Unmarsha/Decode

## CONS

As previously described, I have been learning by implementing this library.
I intend to work on it until I'm satisfied, but unless you are the type of
person who likes to live on the bleeding edge, you probably want to use another library.

# BENCHMARKS

Current status

```
$ go test -tags bench -v -run=none -benchmem -bench .
BenchmarkEncodeFloat32/___lestrrat/float32_via_Encode()-4                     30000000        51.0 ns/op         4 B/op         1 allocs/op
BenchmarkEncodeFloat32/___lestrrat/float32_via_EncodeFloat32()-4             100000000        22.4 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeFloat32/vmihailenco/float32_via_Encode()-4                     20000000        55.8 ns/op         4 B/op         1 allocs/op
BenchmarkEncodeFloat32/vmihailenco/float32_via_EncodeFloat32()-4             100000000        23.9 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeFloat64/___lestrrat/float64_via_Encode()-4                     30000000        57.7 ns/op         8 B/op         1 allocs/op
BenchmarkEncodeFloat64/___lestrrat/float64_via_EncodeFloat64()-4             100000000        25.6 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeFloat64/vmihailenco/float64_via_Encode()-4                     20000000        60.4 ns/op         8 B/op         1 allocs/op
BenchmarkEncodeFloat64/vmihailenco/float64_via_EncodeFloat64()-4              50000000        25.7 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeUint8/___lestrrat/uint8_via_Encode()-4                         30000000        43.8 ns/op         1 B/op         1 allocs/op
BenchmarkEncodeUint8/___lestrrat/uint8_via_EncodeUint8()-4                   100000000        22.7 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeUint8/vmihailenco/uint8_via_Encode()-4                         10000000         178 ns/op         1 B/op         1 allocs/op
BenchmarkEncodeUint8/vmihailenco/uint8_via_EncodeUint8()-4                    50000000        26.9 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeUint16/___lestrrat/uint16_via_Encode()-4                       20000000        54.0 ns/op         2 B/op         1 allocs/op
BenchmarkEncodeUint16/___lestrrat/uint16_via_EncodeUint16()-4                100000000        24.6 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeUint16/vmihailenco/uint16_via_Encode()-4                        5000000         210 ns/op         2 B/op         1 allocs/op
BenchmarkEncodeUint16/vmihailenco/uint16_via_EncodeUint16()-4                 50000000        27.1 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeUint32/___lestrrat/uint32_via_Encode()-4                       20000000        56.2 ns/op         4 B/op         1 allocs/op
BenchmarkEncodeUint32/___lestrrat/uint32_via_EncodeUint32()-4                 50000000        23.5 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeUint32/vmihailenco/uint32_via_Encode()-4                       10000000         203 ns/op         4 B/op         1 allocs/op
BenchmarkEncodeUint32/vmihailenco/uint32_via_EncodeUint32()-4                 50000000        49.7 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeUint64/___lestrrat/uint64_via_Encode()-4                       20000000        75.2 ns/op         8 B/op         1 allocs/op
BenchmarkEncodeUint64/___lestrrat/uint64_via_EncodeUint64()-4                 50000000        29.9 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeUint64/vmihailenco/uint64_via_Encode()-4                       20000000        82.9 ns/op         8 B/op         1 allocs/op
BenchmarkEncodeUint64/vmihailenco/uint64_via_EncodeUint64()-4                 50000000        40.8 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeInt8/___lestrrat/int8_via_Encode()-4                           20000000        61.3 ns/op         1 B/op         1 allocs/op
BenchmarkEncodeInt8/___lestrrat/int8_via_EncodeInt8()-4                       30000000        37.8 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeInt8/vmihailenco/int8_via_Encode()-4                            5000000         277 ns/op         2 B/op         2 allocs/op
BenchmarkEncodeInt8/vmihailenco/int8_via_EncodeInt8()-4                       20000000        52.3 ns/op         1 B/op         1 allocs/op
BenchmarkEncodeInt8FixNum/___lestrrat/int8_via_Encode()-4                     20000000        63.2 ns/op         1 B/op         1 allocs/op
BenchmarkEncodeInt8FixNum/___lestrrat/int8_via_EncodeInt8()-4                 50000000        26.7 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeInt8FixNum/vmihailenco/int8_via_Encode()-4                      5000000         210 ns/op         2 B/op         2 allocs/op
BenchmarkEncodeInt8FixNum/vmihailenco/int8_via_EncodeInt8()-4                 50000000        35.1 ns/op         1 B/op         1 allocs/op
BenchmarkEncodeInt16/___lestrrat/int16_via_Encode()-4                         30000000        50.4 ns/op         2 B/op         1 allocs/op
BenchmarkEncodeInt16/___lestrrat/int16_via_EncodeInt16()-4                   100000000        23.4 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeInt16/vmihailenco/int16_via_Encode()-4                         10000000         186 ns/op         2 B/op         1 allocs/op
BenchmarkEncodeInt16/vmihailenco/int16_via_EncodeInt16()-4                    50000000        44.7 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeInt32/___lestrrat/int32_via_Encode()-4                         20000000        67.5 ns/op         4 B/op         1 allocs/op
BenchmarkEncodeInt32/___lestrrat/int32_via_EncodeInt32()-4                    50000000        33.0 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeInt32/vmihailenco/int32_via_Encode()-4                          5000000         301 ns/op         4 B/op         1 allocs/op
BenchmarkEncodeInt32/vmihailenco/int32_via_EncodeInt32()-4                    50000000        32.1 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeInt64/___lestrrat/int64_via_Encode()-4                         30000000        56.9 ns/op         8 B/op         1 allocs/op
BenchmarkEncodeInt64/___lestrrat/int64_via_EncodeInt64()-4                   100000000        24.8 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeInt64/vmihailenco/int64_via_Encode()-4                         20000000        62.3 ns/op         8 B/op         1 allocs/op
BenchmarkEncodeInt64/vmihailenco/int64_via_EncodeInt64()-4                    50000000        30.6 ns/op         0 B/op         0 allocs/op
BenchmarkEncodeString/___lestrrat/string_(255_bytes)_via_Encode()-4           10000000         208 ns/op       272 B/op         2 allocs/op
BenchmarkEncodeString/___lestrrat/string_(255_bytes)_via_EncodeString()-4     10000000         149 ns/op       256 B/op         1 allocs/op
BenchmarkEncodeString/___lestrrat/string_(256_bytes)_via_Encode()-4           10000000         190 ns/op       272 B/op         2 allocs/op
BenchmarkEncodeString/___lestrrat/string_(256_bytes)_via_EncodeString()-4     10000000         153 ns/op       256 B/op         1 allocs/op
BenchmarkEncodeString/___lestrrat/string_(65536_bytes)_via_Encode()-4           100000       13031 ns/op     65552 B/op         2 allocs/op
BenchmarkEncodeString/___lestrrat/string_(65536_bytes)_via_EncodeString()-4     100000       13420 ns/op     65536 B/op         1 allocs/op
BenchmarkEncodeString/vmihailenco/string_(255_bytes)_via_Encode()-4           10000000         204 ns/op       272 B/op         2 allocs/op
BenchmarkEncodeString/vmihailenco/string_(255_bytes)_via_EncodeString()-4     10000000         148 ns/op       256 B/op         1 allocs/op
BenchmarkEncodeString/vmihailenco/string_(256_bytes)_via_Encode()-4           10000000         206 ns/op       272 B/op         2 allocs/op
BenchmarkEncodeString/vmihailenco/string_(256_bytes)_via_EncodeString()-4     10000000         163 ns/op       256 B/op         1 allocs/op
BenchmarkEncodeString/vmihailenco/string_(65536_bytes)_via_Encode()-4           100000       14108 ns/op     65552 B/op         2 allocs/op
BenchmarkEncodeString/vmihailenco/string_(65536_bytes)_via_EncodeString()-4     100000       19093 ns/op     65536 B/op         1 allocs/op
BenchmarkEncodeArray/___lestrrat/array_via_Encode()-4                          3000000         484 ns/op        56 B/op         4 allocs/op
BenchmarkEncodeArray/___lestrrat/array_via_EncodeArray()-4                     5000000         345 ns/op        56 B/op         4 allocs/op
BenchmarkEncodeArray/vmihailenco/array_via_Encode()-4                          2000000         757 ns/op        33 B/op         2 allocs/op
BenchmarkEncodeMap/___lestrrat/map_via_Encode()-4                              2000000         994 ns/op       208 B/op         8 allocs/op
BenchmarkEncodeMap/___lestrrat/map_via_EncodeMap()-4                           1000000        1092 ns/op       208 B/op         8 allocs/op
BenchmarkEncodeMap/vmihailenco/map_via_Encode()-4                              1000000        1995 ns/op       224 B/op        11 allocs/op
BenchmarkDecodeUint8/___lestrrat/uint8_via_DecodeUint8()-4                    20000000        60.0 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeUint8/vmihailenco/uint8_via_DecodeUint8()_(return)-4           30000000        54.2 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeUint16/___lestrrat/uint16_via_DecodeUint16()-4                 30000000        53.6 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeUint16/vmihailenco/uint16_via_DecodeUint16()_(return)-4        20000000        94.0 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeUint32/___lestrrat/uint32_via_DecodeUint32()-4                 30000000        45.7 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeUint32/vmihailenco/uint32_via_DecodeUint32()_(return)-4        20000000        92.8 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeUint64/___lestrrat/uint64_via_DecodeUint64()-4                 30000000        48.5 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeUint64/vmihailenco/uint64_via_DecodeUint64()_(return)-4        20000000        86.1 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeInt8FixNum/___lestrrat/int8_via_DecodeInt8()-4                 50000000        34.8 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeInt8FixNum/vmihailenco/int8_via_DecodeInt8()_(return)-4        30000000        42.2 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeInt8/___lestrrat/int8_via_DecodeInt8()-4                       30000000        48.6 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeInt8/vmihailenco/int8_via_DecodeInt8()_(return)-4              30000000        55.5 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeInt16/___lestrrat/int16_via_DecodeInt16()-4                    30000000        47.2 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeInt16/vmihailenco/int16_via_DecodeInt16()_(return)-4           20000000        87.8 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeInt32/___lestrrat/int32_via_DecodeInt32()-4                    30000000        44.3 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeInt32/vmihailenco/int32_via_DecodeInt32()_(return)-4           20000000        87.5 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeInt64/___lestrrat/int64_via_DecodeInt64()-4                    30000000        47.6 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeInt64/vmihailenco/int64_via_DecodeInt64()_(return)-4           20000000        84.3 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeFloat32/___lestrrat/float32_via_DecodeFloat32()-4              30000000        39.1 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeFloat32/vmihailenco/float32_via_DecodeFloat32()_(return)-4     20000000        82.0 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeFloat64/___lestrrat/float64_via_DecodeFloat64()-4              30000000        38.4 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeFloat64/vmihailenco/float64_via_DecodeFloat64()_(return)-4     20000000        84.7 ns/op         0 B/op         0 allocs/op
BenchmarkDecodeString/___lestrrat/string_via_Decode()-4                        5000000         294 ns/op       256 B/op         1 allocs/op
BenchmarkDecodeString/___lestrrat/string_via_DecodeString()-4                  5000000         288 ns/op       256 B/op         1 allocs/op
BenchmarkDecodeString/vmihailenco/string_via_Decode()-4                       10000000         223 ns/op       256 B/op         1 allocs/op
BenchmarkDecodeString/vmihailenco/string_via_DecodeString()_(return)-4        10000000         211 ns/op       256 B/op         1 allocs/op
BenchmarkDecodeArray/___lestrrat/array_via_Decode()_(concrete)-4               2000000         773 ns/op       104 B/op         5 allocs/op
BenchmarkDecodeArray/___lestrrat/array_via_Decode()_(interface{})-4            2000000         942 ns/op       120 B/op         6 allocs/op
BenchmarkDecodeArray/___lestrrat/array_via_DecodeArray()-4                     2000000         763 ns/op       104 B/op         5 allocs/op
BenchmarkDecodeArray/vmihailenco/array_via_Decode()_(concrete)-4               1000000        1607 ns/op       248 B/op         9 allocs/op
BenchmarkDecodeArray/vmihailenco/array_via_Decode()_(interface{})-4            2000000         727 ns/op       120 B/op         6 allocs/op
BenchmarkDecodeMap/___lestrrat/map_via_Decode()-4                              1000000        1660 ns/op       440 B/op        12 allocs/op
BenchmarkDecodeMap/___lestrrat/map_via_DecodeMap()-4                           1000000        1609 ns/op       440 B/op        12 allocs/op
BenchmarkDecodeMap/vmihailenco/map_via_Decode()-4                              1000000        1070 ns/op       392 B/op         9 allocs/op
PASS
ok    github.com/lestrrat/go-msgpack  171.139s
```

# ACKNOWLEDGEMENTS

Much has been stolen from https://github.com/vmihailenco/msgpack