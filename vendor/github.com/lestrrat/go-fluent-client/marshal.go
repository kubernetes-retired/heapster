package fluent

import (
	msgpack "github.com/lestrrat/go-msgpack"
)

type marshalFunc func(*Message) ([]byte, error)

func (f marshalFunc) Marshal(msg *Message) ([]byte, error) {
	return f(msg)
}

func msgpackMarshal(m *Message) ([]byte, error) {
	return msgpack.Marshal(m)
}

func jsonMarshal(m *Message) ([]byte, error) {
	return m.MarshalJSON()
}
