// Code specific to deserializing JSON.
package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/pasqal-io/godasse/deserialize/internal"
)

// The deserialization driver for JSON.
type Driver struct{}

// The type of a JSON/Dictionary.
var dictionary = reflect.TypeOf(make(internal.Dict, 0))

// The interface for `json.Unmarshaler`.
var unmarshaler = reflect.TypeOf(new(json.Unmarshaler)).Elem()

// Determine whether we should call the driver to unmarshal values
// of this type from []byte.
//
// For JSON, this is the case if:
// - `typ` represents a dictionary; and/or
// - `typ` implements `json.Unmarshaler`.
//
// You probably won't ever need to call this method.
func (u Driver) ShouldUnmarshal(typ reflect.Type) bool {
	if typ.ConvertibleTo(dictionary) {
		return true
	}
	if reflect.PointerTo(typ).ConvertibleTo(unmarshaler) {
		return true
	}
	return false
}

// Perform unmarshaling.
//
// You probably won't ever need to call this method.
func (u Driver) Unmarshal(buf []byte, out *any) (err error) {
	defer func() {
		// Attempt to intercept errors that leak implementation details.
		if err != nil {
			unmarshalErr := json.UnmarshalTypeError{} //nolint:exhaustruct
			if errors.Is(err, &unmarshalErr) {
				// Go error will mention `map[string] interface{}`, which is an implementation detail.
				err = fmt.Errorf("at %s, invalid json value", unmarshalErr.Field)
			}
		}
	}()

	// Attempt de deserialize as a dictionary.
	if dict, ok := (*out).(*internal.Dict); ok {
		return json.Unmarshal(buf, &dict) //nolint:wrapcheck
	}

	// Attempt to deserialize as a `json.Unmarshaler`.
	if unmarshal, ok := (*out).(json.Unmarshaler); ok {
		return unmarshal.UnmarshalJSON(buf) //nolint:wrapcheck
	}
	return fmt.Errorf("this type cannot be deserialized")
}

var _ internal.Driver = Driver{} // Type assertion.
