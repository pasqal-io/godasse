package shared

import "reflect"

// A value in a dictionary.
//
// We use this type instead of raw type conversions to decrease the risk
// of confusion whenever manipulating `any` and to allow us to work with
// cases in which we do not have direct access to a dictionary.
type Value interface {
	AsDict() (Dict, bool)
	AsSlice() ([]Value, bool)
	Interface() any
}

// A dictionary.
//
// We use this type instead of raw type conversions to decrease the risk
// of confusion whenever manipulating `any` and to allow us to work with
// cases in which we do not have direct access to a dictionary.
type Dict interface {
	Lookup(key string) (Value, bool)
	AsValue() Value
	Keys() []string
}

// A driver for a specific type of deserialization.
type Driver interface {
	// Return true if we have a specific implementation of deserialization
	// for a given type, for instance, if that type implements a specific
	// deserialization interface.
	ShouldUnmarshal(reflect.Type) bool

	Dict([]byte) (Dict, error)

	// Perform unmarshaling for a value.
	Unmarshal([]byte, *any) error

	// Wrap a basic value as a `Value`.
	WrapValue(any) Value
}
