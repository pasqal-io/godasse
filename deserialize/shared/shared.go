package shared

import (
	"reflect"
	"strconv"
)

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

	// Perform unmarshaling for a value.
	Unmarshal(any, *any) error

	// Wrap a basic value as a `Value`.
	WrapValue(any) Value
}

// A parser for strings into primitive values.
type Parser func(source string) (any, error)

func LookupParser(fieldType reflect.Type) *Parser {
	var result *Parser
	switch fieldType.Kind() {
	case reflect.Bool:
		var p Parser = func(source string) (any, error) {
			return strconv.ParseBool(source) //nolint:wrapcheck
		}
		result = &p
	case reflect.Float32:
		var p Parser = func(source string) (any, error) {
			return strconv.ParseFloat(source, 32) //nolint:wrapcheck
		}
		result = &p
	case reflect.Float64:
		var p Parser = func(source string) (any, error) {
			return strconv.ParseFloat(source, 64) //nolint:wrapcheck
		}
		result = &p
	case reflect.Int:
		var p Parser = func(source string) (any, error) {
			return strconv.Atoi(source) //nolint:wrapcheck
		}
		result = &p
	case reflect.Int8:
		var p Parser = func(source string) (any, error) {
			return strconv.ParseInt(source, 10, 8) //nolint:wrapcheck
		}
		result = &p
	case reflect.Int16:
		var p Parser = func(source string) (any, error) {
			return strconv.ParseInt(source, 10, 16) //nolint:wrapcheck
		}
		result = &p
	case reflect.Int32:
		var p Parser = func(source string) (any, error) {
			return strconv.ParseInt(source, 10, 32) //nolint:wrapcheck
		}
		result = &p
	case reflect.Int64:
		var p Parser = func(source string) (any, error) {
			return strconv.ParseInt(source, 10, 64) //nolint:wrapcheck
		}
		result = &p
	case reflect.Uint:
		var p Parser = func(source string) (any, error) {
			// `uint` size is not specified, we'll try with 64 bits.
			return strconv.ParseUint(source, 10, 64) //nolint:wrapcheck
		}
		result = &p
	case reflect.Uint8:
		var p Parser = func(source string) (any, error) {
			return strconv.ParseUint(source, 10, 8) //nolint:wrapcheck
		}
		result = &p
	case reflect.Uint16:
		var p Parser = func(source string) (any, error) {
			return strconv.ParseUint(source, 10, 16) //nolint:wrapcheck
		}
		result = &p
	case reflect.Uint32:
		var p Parser = func(source string) (any, error) {
			return strconv.ParseUint(source, 10, 32) //nolint:wrapcheck
		}
		result = &p
	case reflect.Uint64:
		var p Parser = func(source string) (any, error) {
			return strconv.ParseUint(source, 10, 64) //nolint:wrapcheck
		}
		result = &p
	case reflect.String:
		var p Parser = func(source string) (any, error) {
			return source, nil
		}
		result = &p
	default:
		return nil
	}
	return result
}

// A type that can be deserialized from a shared.Dict.
type UnmarshalDict interface {
	UnmarshalDict(Dict) error
}
