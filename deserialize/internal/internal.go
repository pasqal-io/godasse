package internal

import "reflect"

// A dictionary.
type Dict = map[string]any

// A driver for a specific type of deserialization.
type Driver interface {
	// Return true if we have a specific implementation of deserialization
	// for a given type, for instance, if that type implements a specific
	// deserialization interface.
	ShouldUnmarshal(reflect.Type) bool

	// Perform unmarshaling for a value.
	Unmarshal([]byte, *any) error
}
