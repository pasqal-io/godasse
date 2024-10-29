// Code specific to deserializing JSON.
package json

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/pasqal-io/godasse/deserialize/shared"
)

// The deserialization driver for JSON.
type driver struct{}

func Driver() shared.Driver {
	return driver{}
}

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
	case nil:
		var json JSON = map[string]any{}
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

var _ shared.Value = Value{} //nolint:exhaustruct

func (json JSON) Lookup(key string) (shared.Value, bool) {
	if val, ok := json[key]; ok {
		value := Value{
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
func (json JSON) Keys() []string {
	keys := make([]string, 0)
	for k := range json {
		keys = append(keys, k)
	}
	return keys
}

var _ shared.Dict = JSON{} //nolint:exhaustruct

// The type of a JSON/Dictionary.
var dictionary = reflect.TypeOf(make(JSON, 0))

// The interface for `json.Unmarshaler`.
var unmarshaler = reflect.TypeOf(new(json.Unmarshaler)).Elem()
var textUnmarshaler = reflect.TypeOf(new(encoding.TextUnmarshaler)).Elem()

// Determine whether we should call the driver to unmarshal values
// of this type from []byte.
//
// For JSON, this is the case if:
// - `typ` represents a dictionary; and/or
// - `typ` implements `json.Unmarshaler`.
//
// You probably won't ever need to call this method.
func (driver) ShouldUnmarshal(typ reflect.Type) bool {
	if typ.ConvertibleTo(dictionary) {
		return true
	}
	ptr := reflect.PointerTo(typ)
	return ptr.ConvertibleTo(unmarshaler) || ptr.ConvertibleTo(textUnmarshaler)
}

// Perform unmarshaling.
//
// You probably won't ever need to call this method.
func (u driver) Unmarshal(in any, out *any) (err error) {
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

	var buf []byte
	switch typed := in.(type) {
	// Normalize string, []byte into []byte.
	case string:
		buf = []byte(typed)
	case []byte:
		buf = typed
	// Unwrap Value.
	case Value:
		return u.Unmarshal(typed.wrapped, out)
	case JSON:
		if reflect.TypeOf(out).Elem() == dictionary {
			*out = typed
			return nil
		}

		// Sadly, at this stage, we need to reserialize.
		buf, err = json.Marshal(typed)
		if err != nil {
			return fmt.Errorf("internal error while deserializing: \n\t * %w", err)
		}
	default:
		return fmt.Errorf("expected a string, got %s", in)
	}

	// Attempt to deserialize as a `json.Unmarshaler`.
	if unmarshal, ok := (*out).(json.Unmarshaler); ok {
		err = unmarshal.UnmarshalJSON(buf)
	} else {
		err = json.Unmarshal(buf, out)
	}
	if err == nil {
		// Basic JSON decoding worked, let's go with it.
		return nil
	}
	// But sometimes, things aren't that nice. For instance, time.Time serializes
	// itself as an unencoded string, but its UnmarshalJSON expects an encoded string.
	// Just in case, let's try again with UnmarshalText.
	if textUnmarshaler, ok := (*out).(encoding.TextUnmarshaler); ok {
		err2 := textUnmarshaler.UnmarshalText(buf)
		if err2 == nil {
			// Success! Let's use that result.
			return nil
		}
		return fmt.Errorf("failed to unmarshal '%s' either from JSON or from text: \n\t * %w\n\t * and %w", buf, err, err2)
	}
	return fmt.Errorf("failed to unmarshal '%s': \n\t * %w", buf, err)
}

func (driver) WrapValue(wrapped any) shared.Value {
	return Value{
		wrapped: wrapped,
	}
}

func (driver) Enter(string, reflect.Type) error {
	// No particular protocol to follow.
	return nil
}
func (driver) Exit(reflect.Type) {
	// No particular protocol to follow.
}

var _ shared.Driver = driver{} // Type assertion.
