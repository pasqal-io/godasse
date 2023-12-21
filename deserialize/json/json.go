// Code specific to deserializing JSON.
package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/pasqal-io/godasse/deserialize/shared"
)

// The deserialization driver for JSON.
type Driver struct{}

// A JSON value.
type Value struct {
	wrapped any
}

// A JSON object.
type JSON map[string]any

func (v Value) AsDict() (shared.Dict, bool) {
	switch t := v.wrapped.(type) {
	case JSON:
		return t, true
	case map[string]any:
		var json JSON = t
		return json, true
	default:
		return nil, false
	}
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
func (v Value) Interface() any {
	return v.wrapped
}

var _ shared.Value = Value{}

func (json JSON) Lookup(key string) (shared.Value, bool) {
	if val, ok := json[key]; ok {
		var value Value = Value{
			wrapped: val,
		}
		return value, true
	}
	return nil, false
}
func (json JSON) AsValue() shared.Value {
	return Value{
		wrapped: json,
	}
}

var _ shared.Dict = JSON{}

// The type of a JSON/Dictionary.
var dictionary = reflect.TypeOf(make(JSON, 0))

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

// Deserialize to a JSON dict.
//
// You probably won't ever need to call this method.
func (u Driver) Dict(buf []byte) (result shared.Dict, err error) {
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
	dict := make(JSON)
	err = json.Unmarshal(buf, &dict)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	return dict, nil
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
	if dict, ok := (*out).(*shared.Dict); ok {
		return json.Unmarshal(buf, &dict) //nolint:wrapcheck
	}

	// Attempt to deserialize as a `json.Unmarshaler`.
	if unmarshal, ok := (*out).(json.Unmarshaler); ok {
		return unmarshal.UnmarshalJSON(buf) //nolint:wrapcheck
	}
	return fmt.Errorf("type %s cannot be deserialized", reflect.TypeOf(out).Name())
}

func (u Driver) WrapValue(wrapped any) shared.Value {
	return Value{
		wrapped: wrapped,
	}
}

var _ shared.Driver = Driver{} // Type assertion.
