package kvlist

import (
	"encoding"
	"fmt"
	"reflect"

	"github.com/pasqal-io/godasse/deserialize/shared"
)

// The deserialization driver for (k, value list).
type Driver struct{}

// The type of a (key, value list) store.
type KVList map[string][]string

type Value struct {
	wrapped any
}

// A KVValue may never be converted into a string.
func (v Value) AsDict() (shared.Dict, bool) {
	return nil, false
}
func (v Value) Interface() any {
	return v
}
func (v Value) AsSlice() ([]shared.Value, bool) {
	if wrapped, ok := v.wrapped.([]any); ok {
		result := make([]shared.Value, len(wrapped))
		for i, value := range wrapped {
			result[i] = Value{wrapped: value}
		}
		return result, true
	}
	return nil, false
}

var _ shared.Value = Value{} //nolint:exhaustruct

func (list KVList) Lookup(key string) (shared.Value, bool) {
	if val, ok := list[key]; ok {
		return Value{
			wrapped: val,
		}, true
	}
	return nil, false
}
func (list KVList) AsValue() shared.Value {
	return Value{
		wrapped: list,
	}
}
func (list KVList) Keys() []string {
	keys := make([]string, 0)
	for k := range list {
		keys = append(keys, k)
	}
	return keys
}

var _ shared.Dict = make(KVList, 0)

// A type that supports deserialization from bytes.
type Unmarshaler interface {
	Unmarshal([]byte) error
}

// The type KVList.
var kvList = reflect.TypeOf(make(KVList, 0))

// The interface `TextUnmarshaler`.
var textUnmarshaler = reflect.TypeOf(new(encoding.TextUnmarshaler)).Elem()

// Determine whether we should call the driver to unmarshal values
// of this type from []byte.
//
// For KVList, this is the case if:
// - `typ` represents a KVList; and/or
// - `typ` implements `Unmarshaler`.
func (u Driver) ShouldUnmarshal(typ reflect.Type) bool {
	if typ.ConvertibleTo(kvList) {
		return true
	}
	if typ.Implements(textUnmarshaler) || reflect.PointerTo(typ).Implements(textUnmarshaler) {
		return true
	}
	if typ.ConvertibleTo(textUnmarshaler) {
		return true
	}
	return false
}

// Perform unmarshaling.
func (u Driver) Unmarshal(in any, out *any) (err error) {
	var buf []byte
	switch typed := in.(type) {
	case string:
		buf = []byte(typed)
	case []byte:
		buf = typed
	case []string:
		if len(typed) == 1 {
			buf = []byte(typed[0])
		} else {
			return fmt.Errorf("cannot deserialize []string in this context")
		}
	case Value:
		return u.Unmarshal(typed.wrapped, out)
	case KVList:
		if reflect.TypeOf(out).Elem() == kvList {
			*out = typed
			return nil
		}
		return fmt.Errorf("cannot deserialize map[string][]string in this context")
	default:
		return fmt.Errorf("expected a string, got %s", in)
	}

	if unmarshal, ok := (*out).(encoding.TextUnmarshaler); ok {
		return unmarshal.UnmarshalText(buf) //nolint:wrapcheck
	}
	return fmt.Errorf("this type cannot be deserialized")
}

func (u Driver) WrapValue(wrapped any) shared.Value {
	return Value{
		wrapped: wrapped,
	}
}

var _ shared.Driver = Driver{} //nolint:exhaustruct
