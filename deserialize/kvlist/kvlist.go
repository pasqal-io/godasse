package kvlist

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"

	"github.com/pasqal-io/godasse/deserialize/shared"
)

// The deserialization driver for (k, value list).
//
// The fields of this value are used only while building the deserializer.
type driver struct {
	// If non-nil, we have entered the root struct while building the deserializer
	// and this points to the type of the root struct.
	enteredStructAt *reflect.Type

	// If non-nil, we have entered a slice or array field within the root struct
	// while building the deserializer and this points to the type of the slice or
	// array field.
	enteredSliceAt *reflect.Type

	// If non-nil, we have entered a leaf, i.e. either the contents of the slice or
	// array field within the root struct or a data structure that supports `TextUnmarshaler`,
	// while building the deserializer and this points to the type of the slice or array field.
	enteredLeafAt *reflect.Type
}

func Driver() shared.Driver {
	return &driver{
		enteredStructAt: nil,
		enteredSliceAt:  nil,
		enteredLeafAt:   nil,
	}
}

// The type of a (key, value list) store.
type KVList map[string][]string

type dict struct {
	wrapped map[string]any
}

func MakeRootDict(wrapped map[string]any) shared.Dict {
	return dict{wrapped}
}

func (d dict) Lookup(key string) (shared.Value, bool) {
	v, ok := d.wrapped[key]
	if !ok {
		return Value{nil}, false
	}
	return Value{v}, true

}

func (d dict) AsValue() shared.Value {
	return Value{
		wrapped: d.wrapped,
	}
}

func (d dict) Keys() []string {
	keys := []string{}
	for k := range d.wrapped {
		keys = append(keys, k)
	}
	return keys
}

type Value struct {
	wrapped any
}

func (v Value) AsDict() (shared.Dict, bool) {
	if asDict, ok := v.wrapped.(map[string]any); ok {
		return dict{
			wrapped: asDict,
		}, true
	}
	return nil, false
}
func (v Value) Interface() any {
	return v.wrapped
}
func (v Value) AsSlice() ([]shared.Value, bool) {
	if wrapped, ok := v.wrapped.([]any); ok {
		result := make([]shared.Value, len(wrapped))
		for i, value := range wrapped {
			result[i] = Value{wrapped: value}
		}
		return result, true
	}
	if wrapped, ok := v.wrapped.([]string); ok {
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
func (u *driver) ShouldUnmarshal(typ reflect.Type) bool {
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
func (u *driver) Unmarshal(in any, out *any) (err error) {
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
			return errors.New("cannot deserialize []string in this context")
		}
	case Value:
		return u.Unmarshal(typed.wrapped, out)
	case KVList:
		if reflect.TypeOf(out).Elem() == kvList {
			*out = typed
			return nil
		}
		return errors.New("cannot deserialize map[string][]string in this context")
	default:
		return fmt.Errorf("expected a string, got %s", in)
	}

	if unmarshal, ok := (*out).(encoding.TextUnmarshaler); ok {
		return unmarshal.UnmarshalText(buf) //nolint:wrapcheck
	}
	return errors.New("this type cannot be deserialized")
}

func (u *driver) WrapValue(wrapped any) shared.Value {
	return Value{
		wrapped: wrapped,
	}
}

func canBeALeaf(typ reflect.Type) bool {
	switch typ.Kind() {
	// Primitive-ish types that can be trivially parsed.
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		fallthrough
	case reflect.Bool:
		fallthrough
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		fallthrough
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		fallthrough
	case reflect.String:
		return true
	default:
		// Types that can be unmarshaled.
		return typ.Implements(textUnmarshaler) || typ.ConvertibleTo(textUnmarshaler) || reflect.PointerTo(typ).Implements(textUnmarshaler) || reflect.PointerTo(typ).ConvertibleTo(textUnmarshaler)
	}
}

func (u *driver) Enter(at string, typ reflect.Type) error {
	kind := typ.Kind()
	if kind == reflect.Pointer {
		// ignore pointers entirely
		return nil
	}
	switch {
	// Initial state.
	case u.enteredStructAt == nil:
		if kind != reflect.Struct {
			return fmt.Errorf("KVList deserialization expects a struct, got %s", typ.String())
		}
		u.enteredStructAt = &typ
	case u.enteredSliceAt == nil && u.enteredLeafAt == nil:
		if u.enteredStructAt == nil {
			panic("internal error: inconsistent state")
		}
		switch {
		case canBeALeaf(typ):
			u.enteredLeafAt = &typ
		case kind == reflect.Array || kind == reflect.Slice:
			u.enteredSliceAt = &typ
		default:
			return fmt.Errorf("KVList deserialization expects a struct of slices of trivially deserializable types, but at %s, got %s", at, typ.String())
		}
	case u.enteredLeafAt == nil && u.enteredSliceAt != nil:
		if u.enteredStructAt == nil {
			panic("internal error: inconsistent state")
		}
		if canBeALeaf(typ) {
			u.enteredSliceAt = &typ
		} else {
			return fmt.Errorf("KVList deserialization expects a struct of slices of trivially deserializable types, but at %s, got %s", at, typ.String())
		}
	default:
		if u.enteredStructAt == nil || u.enteredLeafAt == nil {
			panic("internal error: inconsistent state")
		}
		// We're in a leaf, there isn't anything we can check.
	}

	return nil
}

func (u *driver) Exit(typ reflect.Type) {
	kind := typ.Kind()
	if kind == reflect.Pointer {
		// ignore pointers entirely
		return
	}
	switch {
	case u.enteredLeafAt != nil && *u.enteredLeafAt == typ:
		u.enteredLeafAt = nil
	case u.enteredSliceAt != nil && *u.enteredSliceAt == typ:
		u.enteredSliceAt = nil
	case u.enteredStructAt != nil && *u.enteredStructAt == typ:
		u.enteredStructAt = nil
	}
}

var _ shared.Driver = &driver{} //nolint:exhaustruct
