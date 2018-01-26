package msgpack

import (
	"reflect"
	"sync"

	"github.com/pkg/errors"
)

var muExtDecode sync.RWMutex
var muExtEncode sync.RWMutex
var extDecodeRegistry = make(map[int]reflect.Type)
var extEncodeRegistry = make(map[reflect.Type]int)

var decodeMsgpackerType = reflect.TypeOf((*DecodeMsgpacker)(nil)).Elem()
var encodeMsgpackerType = reflect.TypeOf((*EncodeMsgpacker)(nil)).Elem()

func RegisterExt(typ int, v interface{}) error {
	rt := reflect.TypeOf(v)

	var decodeType = rt
	var encodeType = rt
	if decodeType.Implements(decodeMsgpackerType) {
		if decodeType.Kind() == reflect.Ptr {
			decodeType = decodeType.Elem()
		}
	} else {
		if ptrT := reflect.PtrTo(decodeType); !ptrT.Implements(decodeMsgpackerType) {
			return errors.Errorf(`msgpack: invalid type %s: only DecodeMsgpackers can be registered`, decodeType)
		}
	}

	if !encodeType.Implements(encodeMsgpackerType) {
		if encodeType.Kind() == reflect.Ptr && encodeType.Elem().Implements(encodeMsgpackerType) {
			encodeType = encodeType.Elem()
		} else {
			return errors.Errorf(`msgpack: invalid type %s: only EncodeMsgpackers can be registered`, encodeType)
		}
	}

	muExtDecode.Lock()
	extDecodeRegistry[typ] = decodeType
	muExtDecode.Unlock()

	muExtEncode.Lock()
	extEncodeRegistry[encodeType] = typ
	muExtEncode.Unlock()

	return nil
}
