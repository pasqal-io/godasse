// Mechanisms to deal with initialization and validation of values.
//
// These interfaces are primarily designed to be implemented by
// deserialization schemas.
package validation

import (
	"fmt"
	"reflect"
)

// A type that supports initialization.
//
// Our deserialization library automatically runs any call to `Initialize()`
// at every depth of the tree, **before** building the node.
//
// Important: We expect `Initializer` to be implemented on **pointers**,
// rather than on structs.
//
// Otherwise, all its operations are performed on a copy of the struct and
// the result is lost immediately.
type Initializer interface {
	// Setup the contents of the struct.
	Initialize() error
}

// A type that supports validation.
//
// Our deserialization library automatically runs any call to `Validate()`,
// at every depth of the tree, **after** building the node.
//
// Important: We expect `Validator` to be implemented on **pointers**,
// rather than on structs.
//
// This lets `Validate()` perform any necessary changes to the data
// structure. In particular, if necessary, it may be used to populate
// private fields from the contents of public fields.
type Validator interface {
	// Confirm that the data is valid.
	//
	// Return an error if it is invalid.
	//
	// If necessary, this method may alter the contents of the struct.
	Validate() error
}

// A validation error.
//
// Use errors.As() or Unwrap() to expose the error returned by Validate().
type Error struct {
	// Where the validation error happened.
	path *path

	// The error returned by `Validate()`.
	wrapped error
}

// Extract a human-readable string.
func (v Error) Error() string {
	buf := []string{}
	cursor := v.path
	for cursor != nil {
		switch cursor.kind {
		case kindField:
			buf = append(buf, fmt.Sprint(cursor.entry))
		case kindIndex:
			buf = append(buf, fmt.Sprintf("[%d]", cursor.entry))
		case kindKey:
			buf = append(buf, fmt.Sprintf("[>> %v <<]", cursor.entry))
		case kindValue:
			buf = append(buf, fmt.Sprintf("[%v]", cursor.entry))
		case kindInterface:
			// Keep buf unchanged.
		case kindDereference:
			// Keep buf unchanged.
		}
		cursor = cursor.prev
	}
	serialized := ""
	for i := len(buf) - 1; i >= 0; i-- {
		serialized += buf[i]
	}
	if serialized == "" {
		serialized = "root" //nolint:ineffassign
	}
	return fmt.Sprintf("validation error at %s:\n\t * %s", buf, v.wrapped.Error())
}

// Unwrap the underlying validation error.
func (v Error) Unwrap() error {
	return v.wrapped
}

// A type of entry in a path.
//
// Used to simplify path management.
type entryKind string

const (
	// Visiting a field.
	kindField entryKind = "FIELD"

	// Visiting a slice or array.
	kindIndex entryKind = "INDEX"

	// Visiting an interface.
	kindInterface entryKind = "INTERFACE"

	// Visiting a key within a map.
	kindKey entryKind = "KEY"

	// Visiting a value within a map.
	kindValue entryKind = "VALUE"

	// Dereferencing a pointer.
	kindDereference entryKind = "POINTER"
)

// A path while visiting a data structure.
type path struct {
	// The latest entry in the path, e.g. a field name or an index value.
	entry any

	// The kind of the latest entry in the path.
	kind entryKind

	// The previous entry in the path.
	prev *path
}

func (v *path) push(entry any, kind entryKind) *path {
	result := new(path)
	result.entry = entry
	result.prev = v
	result.kind = kind
	return result
}

func validateReflect(path *path, value reflect.Value) error {
	if !value.IsValid() {
		// We're dealing with the unwrapped nil value, which cannot implement
		// Validator in any way.
		return nil
	}
	switch value.Type().Kind() {
	case reflect.Interface:
		elem := value.Elem()
		err := validateReflect(path.push("", kindInterface), elem)
		if err != nil {
			return err
		}
	case reflect.Array:
		fallthrough
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			err := validateReflect(path.push(i, kindIndex), value.Index(i))
			if err != nil {
				return err
			}
		}
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			k := iter.Key()
			err := validateReflect(path.push(k, kindKey), k)
			if err != nil {
				return err
			}

			err = validateReflect(path.push(k, kindValue), iter.Value())
			if err != nil {
				return err
			}
		}
	case reflect.Pointer:
		err := validateReflect(path.push(0, kindDereference), value.Elem())
		if err != nil {
			return err
		}
	case reflect.Struct:
		reflectedType := value.Type()
		for i := 0; i < value.NumField(); i++ {
			subPath := path.push(reflectedType.Field(i).Name, kindField)
			err := validateReflect(subPath, value.Field(i))
			if err != nil {
				return err
			}
		}

	default:
		// There are no validations for other types.
		break
	}

	if value.CanInterface() { // We cannot call validator on unexported fields. Sigh.
		toValidate := value
		// Validation is implemented on pointers, so we need a pointer.
		if value.Type().Kind() != reflect.Pointer {
			if value.CanAddr() {
				// Lucky us, we can get a pointer.
				toValidate = value.Addr()
			} else {
				// ...but there are many cases that do not support grabbing a pointer, including
				// interfaces, map keys or map values. In these cases, we must make a copy and
				// point to that copy.
				ptrCopy := reflect.New(value.Type())
				ptrCopy.Elem().Set(value)
				toValidate = ptrCopy
			}
		}
		asAny := toValidate.Interface()
		if validator, ok := asAny.(Validator); ok {
			if err := validator.Validate(); err != nil {
				return Error{
					wrapped: err,
					path:    path,
				}
			}
		}
	}
	return nil
}
func Validate[T any](value *T) error {
	reflected := reflect.ValueOf(value)
	return validateReflect(nil, reflected)
}
