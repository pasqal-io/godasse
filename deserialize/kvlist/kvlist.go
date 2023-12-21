package kvlist

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/pasqal-io/godasse/deserialize/internal"
)

// The deserialization driver for (k, value list).
type Driver struct{}

// The type of a (key, value list) store.
type KVList map[string][]string

// A type that supports deserialization from bytes.
type Unmarshaler interface {
	Unmarshal([]byte) error
}

// The type KVList.
var kvList = reflect.TypeOf(make(KVList, 0))

// The interface `Unmarshaler`.
var unmarshaler = reflect.TypeOf(new(json.Unmarshaler)).Elem()

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
	if typ.ConvertibleTo(unmarshaler) {
		return true
	}
	return false
}

// Perform unmarshaling.
func (u Driver) Unmarshal(buf []byte, out *any) (err error) {
	if dict, ok := (*out).(*internal.Dict); ok {
		return json.Unmarshal(buf, &dict) //nolint:wrapcheck
	}
	if unmarshal, ok := (*out).(Unmarshaler); ok {
		return unmarshal.Unmarshal(buf) //nolint:wrapcheck
	}
	return fmt.Errorf("this type cannot be deserialized")
}

var _ internal.Driver = Driver{} // Type assertion.
