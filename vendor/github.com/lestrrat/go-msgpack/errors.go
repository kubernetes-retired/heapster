package msgpack

import "reflect"

func (e *InvalidDecodeError) Error() string {
	if e.Type == nil {
		return "msgpack: Decode(nil)"
	}

	if e.Type.Kind() != reflect.Ptr {
		return "msgpack: Decode(non-pointer " + e.Type.String() + ")"
	}
	return "msgpack: Decode(nil " + e.Type.String() + ")"
}
