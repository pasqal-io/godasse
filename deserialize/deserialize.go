// Out of the box, Go json (and other deserializers) cannot make a difference between
// JSON's `undefined` (a key is missing) and a `0`/`false`/`""`/`[]`. This means that
// code that uses the stdlib's default deserialization is going to inject these zero
// values all over the place if it receives input that is missing fields or that has
// misspelled field names.
//
// This package implements an alternative deserialization.
//
// # Recommended use
//
// If you have a struct `FooSchema` that you wish to deserialize:
//
// - To define default values for fields (in particular private fields), implement `Initializer`
//
//	func (result *FooSchema) Initialize() err {
//		  result.MyField1 = defaultValue1
//	   result.MyField2 = defaultValue2
//	   ...
//	   return err
//	}
//
// - To define a validator, implement `Validator`
//
//			func (result *FooSchema) Validator() err {
//			   if result.MyField1 > 100 {
//	           return fmt.Errorf("invalid value for MyField1!") // The error will be visible to end users.
//		    }
//	     ..
//	     return nil
//		}
//
// (apologies for weird formatting, please blame gofmt)
//
// Same behavior as the standard library:
//   - if a struct implements `json.Unmarshaler`, short-circuit deserialization and use
//     this method instead of anything built-in;
//   - lower-case field names mean that we NEVER accept external data during deserialization;
//   - enforces `json:"XXXX"` renamings when deserializing JSON;
//   - renaming with `json:"XXXX"` will not make a field public;
//   - a field renamed to `json:"-"` will not accept external data during deserialization.
//
// Different behavior:
//   - this library also works for formats other than json, in which case instead of tag `json`,
//     we use a specific tag (e.g. "query" or "path");
//   - if a value implements `Initializer`, we run the initializer before deserializing
//     the value (this is the only way to provide default values for private fields);
//   - if a tag `default:"XXX"` is specified, we use this value when a field is not specified
//     (by opposition, Go would silently insert zero values);
//   - if a tag `orMethod:"XXX"` is specified, we attempt to call the corresponding method
//     when a field is not specified (by opposition, Go would silently insert zero values);
//   - if a tag `initialized:""` is specified, we will not complain
//   - if a data structure supports `Validator`, we run validation during deserialization
//     and fail if validation rejects the value (by opposition, in Go, you need to run any
//     validation step manually, after deserialization completes);
//   - we attempt to detect errors early and fail when setting up the deserializer, instead
//     of ignoring errors and/or failing during deserialization.
//
// # Warning
//
// By design, Go will NOT let us deserialize, validate or apply default values to private
// fields (i.e. fields which start lower-case). This is a decision that goes deep in the
// language. If you have a private field, it will be initialized to its zero value unless
// you implement `Initializer` on the struct containing.
package deserialize

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/pasqal-io/godasse/deserialize/internal"
	jsonPkg "github.com/pasqal-io/godasse/deserialize/json"
	"github.com/pasqal-io/godasse/deserialize/kvlist"
	"github.com/pasqal-io/godasse/deserialize/shared"
	tagsPkg "github.com/pasqal-io/godasse/deserialize/tags"
	"github.com/pasqal-io/godasse/validation"
)

// -------- Public API --------

type Unmarshaler shared.Driver

// Options for building a deserializer.
//
// See also JSONOptions, QueryOptions, etc. for reasonable
// default values.
type Options struct {
	// The name of tags used for renamings (e.g. "json").
	//
	// If you leave this blank, defaults to "json".
	MainTagName string

	// Human-readable information on the nature of data
	// you'll be deserializing with this deserializer.
	//
	// Used for logging and error messages.
	//
	// For instance, if you're deserializing for an endpoint
	// "GET /api/v1/fetch", string "GET /api/v1/fetch" is an
	// acceptable value for RootPath.
	//
	// Optional. If you leave this blank, no human-readable
	// information will be added.
	RootPath string

	// An unmarshaler, used to deserialize values when they
	// are provided as []byte or string.
	Unmarshaler Unmarshaler
}

// The de facto JSON type in Go.

// A preset fit for consuming JSON.
//
// Params:
//   - root A human-readable root (e.g. the name of the endpoint). Used only
//     for error reporting. `""` is a perfectly acceptable root.
func JSONOptions(root string) Options {
	return Options{
		MainTagName: "json",
		RootPath:    root,
		Unmarshaler: jsonPkg.Driver{},
	}
}

// A preset fit for consuming Queries.
//
// Params:
//   - root A human-readable root (e.g. the name of the endpoint). Used only
//     for error reporting. `""` is a perfectly acceptable root.
func QueryOptions(root string) Options {
	return Options{
		MainTagName: "query",
		RootPath:    root,
		Unmarshaler: kvlist.Driver{},
	}
}

// A deserializer from strings or buffers.
type BytesDeserializer[To any] interface {
	DeserializeString(string) (*To, error)
	DeserializeBytes([]byte) (*To, error)
}

// A deserializers from dictionaries
//
// Use this to deserialize e.g. JSON bodies.
type MapDeserializer[To any] interface {
	BytesDeserializer[To]
	// Deserialize a single value from a dict.
	DeserializeDict(shared.Dict) (*To, error)
	// Deserialize a list of values from a list of values.
	DeserializeList([]shared.Value) ([]To, error)
}

// A deserializer from key, lists of values.
//
// Use this to deserialize e.g. query strings.
type KVListDeserializer[To any] interface {
	DeserializeKVList(kvlist.KVList) (*To, error)
}

// Create a deserializer from Dict.
func MakeMapDeserializer[T any](options Options) (MapDeserializer[T], error) {
	tagName := options.MainTagName
	if tagName == "" {
		return nil, errors.New("missing option MainTagName")
	}
	return makeOuterStructDeserializer[T](options.RootPath, staticOptions{
		renamingTagName: tagName,
		allowNested:     true,
		unmarshaler:     options.Unmarshaler,
	})
}

// Create a deserializer from (key, value list).
func MakeKVListDeserializer[T any](options Options) (KVListDeserializer[T], error) {
	tagName := options.MainTagName
	if tagName == "" {
		return nil, errors.New("missing option MainTagName")
	}
	innerOptions := staticOptions{
		renamingTagName: tagName,
		allowNested:     false,
		unmarshaler:     options.Unmarshaler,
	}
	wrapped, err := makeOuterStructDeserializer[T](options.RootPath, innerOptions)
	if err != nil {
		return nil, err
	}
	deserializer := func(value kvlist.KVList) (*T, error) {
		normalized := make(jsonPkg.JSON)
		err := deListMap[T](normalized, value, innerOptions)
		if err != nil {
			return nil, fmt.Errorf("error attempting to deserialize from a list of entries:\n\t * %w", err)
		}
		return wrapped.deserializer(normalized)
	}
	return kvListDeserializer[T]{
		deserializer: deserializer,
		options:      innerOptions,
	}, nil
}

// An error that arises because of a bug in a custom deserializer.
type CustomDeserializerError struct {
	// The operation that failed, e.g. "initialize", "orMethod".
	Operation string

	// The kind of value we were applying it to, e.g. "outer", "struct", "map", "ptr", "field".
	Structure string

	// The underlying error.
	Wrapped error
}

// Return the user-facing message.
func (e CustomDeserializerError) Error() string {
	return e.Wrapped.Error()
}

// Unwrap the error.
func (e CustomDeserializerError) Unwrap() error {
	return e.Wrapped
}

var _ error = CustomDeserializerError{} //nolint:exhaustruct

// ----------------- Private

// Options used while setting up a deserializer.
type staticOptions struct {
	// The name of tag used for renamings (e.g. "json").
	renamingTagName string

	// If true, allow the outer struct to contain arrays, slices and inner structs.
	//
	// Otherwise, the outer struct is only allowed to contain flat types.
	allowNested bool

	unmarshaler Unmarshaler
}

// A deserializer from (key, value) maps.
type mapDeserializer[T any] struct {
	deserializer func(value shared.Dict) (*T, error)
	options      staticOptions
}

func (me mapDeserializer[T]) DeserializeBytes(source []byte) (*T, error) {
	dict := new(any)
	if err := me.options.unmarshaler.Unmarshal(source, dict); err != nil {
		return nil, fmt.Errorf("failed to deserialize source: \n\t * %w", err)
	}
	asDict, ok := me.options.unmarshaler.WrapValue(*dict).AsDict()
	if !ok {
		return nil, fmt.Errorf("failed to deserialize as a dictionary")
	}
	return me.deserializer(asDict)
}

func (me mapDeserializer[T]) DeserializeString(source string) (*T, error) {
	return me.DeserializeBytes([]byte(source))
}

func (me mapDeserializer[T]) DeserializeDict(value shared.Dict) (*T, error) {
	return me.deserializer(value)
}

func (me mapDeserializer[T]) DeserializeList(list []shared.Value) ([]T, error) {
	result := []T{}
	for i, entry := range list {
		if dict, ok := entry.AsDict(); ok {
			r, err := me.deserializer(dict)
			if err != nil {
				return []T{}, fmt.Errorf("failed to deserialize entry %d: \n\t * %w", i, err)
			}
			result = append(result, *r)
		}
	}
	return result, nil
}

// A deserializer from (key, []string) maps.
type kvListDeserializer[T any] struct {
	deserializer func(value kvlist.KVList) (*T, error)
	options      staticOptions
}

func (me kvListDeserializer[T]) DeserializeKVList(value kvlist.KVList) (*T, error) {
	return me.deserializer(value)
}

func (me kvListDeserializer[T]) DeserializeBytes(source []byte) (*T, error) {
	kvList := new(kvlist.KVList)
	if !me.options.unmarshaler.ShouldUnmarshal(reflect.TypeOf(*kvList)) {
		return nil, fmt.Errorf("this deserializer does not support deserializing from bytes")
	}

	var kvListAny any = kvList
	err := me.options.unmarshaler.Unmarshal(source, &kvListAny)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	return me.deserializer(*kvList)
}

func (me kvListDeserializer[T]) DeserializeString(source string) (*T, error) {
	return me.DeserializeBytes([]byte(source))
}

// Convert a `map[string][]string` (as provided e.g. by the query parser) into a `Dict`
// (as consumed by this parsing mechanism).
func deListMap[T any](outMap map[string]any, inMap map[string][]string, options staticOptions) error {
	var fakeValue *T
	reflectedT := reflect.TypeOf(fakeValue).Elem()
	if reflectedT.Kind() != reflect.Struct {
		return fmt.Errorf("cannot implement a MapListDeserializer without a struct, got %s", reflectedT.Name())
	}

	for i := 0; i < reflectedT.NumField(); i++ {
		field := reflectedT.Field(i)
		tags, err := tagsPkg.Parse(field.Tag)
		if err != nil {
			return fmt.Errorf("invalid tags\n\t * %w", err)
		}

		// We'll use the public field name both to fetch from `value` and to write to `out`.
		publicFieldName := tags.PublicFieldName(options.renamingTagName)
		if publicFieldName == nil {
			publicFieldName = &field.Name
		}

		switch field.Type.Kind() {
		case reflect.Array:
			fallthrough
		case reflect.Slice:
			outMap[*publicFieldName] = inMap[*publicFieldName]
		case reflect.Bool:
			fallthrough
		case reflect.String:
			fallthrough
		case reflect.Float32:
			fallthrough
		case reflect.Float64:
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
		case reflect.Uint8:
			fallthrough
		case reflect.Uint16:
			fallthrough
		case reflect.Uint32:
			fallthrough
		case reflect.Uint64:
			length := len(inMap[*publicFieldName])
			switch length {
			case 0: // No value.
			case 1: // One value, we can fit it into a single entry of outMap.
				outMap[*publicFieldName] = inMap[*publicFieldName][0]
			default:
				return fmt.Errorf("cannot fit %d elements into a single entry of field %s.%s", length, reflectedT.Name(), field.Name)
			}
		default:
			panic("This should not happen")
		}
	}
	return nil
}

// A type of deserializers using reflection to perform any conversions.
type reflectDeserializer func(slot *reflect.Value, data shared.Value) error

// The interface `validation.Initializer`, which we use throughout the code
// to pre-initialize structs.
var initializerInterface = reflect.TypeOf((*validation.Initializer)(nil)).Elem()
var validatorInterface = reflect.TypeOf((*validation.Validator)(nil)).Elem()

// The interface `error`.
var errorInterface = reflect.TypeOf((*error)(nil)).Elem()

const JSON = "json"

// Construct a statically-typed deserializer.
//
// Under the hood, this uses the reflectDeserializer.
//
//   - `path` a human-readable path (e.g. the name of the endpoint) or "" if you have nothing
//     useful for human beings;
//   - `tagName` the name of tags to use for field renamings, e.g. `query`.
func makeOuterStructDeserializer[T any](path string, options staticOptions) (*mapDeserializer[T], error) {
	container := new(T) // An uninitialized container, used to extract type information and call initializer methods.

	if options.unmarshaler == nil {
		return nil, errors.New("please specify an unmarshaler")
	}

	// Pre-check if we're going to perform initialization.
	typ := reflect.TypeOf(*container)
	shouldPreinitialize, err := canInterface(typ, initializerInterface)
	if err != nil {
		return nil, err
	}
	if path == "" {
		path = typeName(typ)
	} else {
		path = fmt.Sprint(path, ".", typeName(typ))
	}

	// The outer struct can't have any tags attached.
	tags := tagsPkg.Empty()
	reflectDeserializer, err := makeStructDeserializerFromReflect(path, typ, options, &tags, reflect.ValueOf(container), shouldPreinitialize)
	if err != nil {
		return nil, err
	}

	var result = mapDeserializer[T]{
		deserializer: func(value shared.Dict) (*T, error) {
			result := new(T)
			if shouldPreinitialize {
				var resultAny any = result
				initializer, ok := resultAny.(validation.Initializer)
				var err error
				if !ok {
					err = fmt.Errorf("we have already checked that the result can be converted to `Initializer` but conversion has failed")
					panic(err)
				}
				err = initializer.Initialize()
				if err != nil {
					err = fmt.Errorf("at %s, encountered an error while initializing optional fields:\n\t * %w", path, err)
					slog.Error("internal error during deserialization", "error", err)
					return nil, CustomDeserializerError{
						Wrapped:   err,
						Operation: "initializer",
						Structure: "outer",
					}

				}
			}
			resultSlot := reflect.ValueOf(result).Elem()
			input := value.AsValue()
			err := reflectDeserializer(&resultSlot, input)
			if err != nil {
				return nil, err
			}
			return result, nil
		},
		options: options,
	}
	return &result, nil
}

// Construct a dynamically-typed deserializer for structs.
//
//   - `path` the human-readable path into the data structure, used for error-reporting;
//   - `typ` the dynamic type for the struct being compiled;
//   - `tags` the table of tags for this field.
//   - `wasPreinitialized` if this value was preinitialized, typically through `Initializer`
func makeStructDeserializerFromReflect(path string, typ reflect.Type, options staticOptions, tags *tagsPkg.Tags, container reflect.Value, wasPreInitialized bool) (reflectDeserializer, error) {
	if typ.Kind() != reflect.Struct {
		panic(fmt.Sprintf("invalid call to StructDeserializer: %s is not a struct", path))
	}
	selfContainer := reflect.New(typ)
	deserializers := make(map[string]func(outPtr *reflect.Value, inMap shared.Dict) error)

	initializationData, err := initializationData(path, typ, options)
	if err != nil {
		return nil, err
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldType := field.Type
		tags, err := tagsPkg.Parse(field.Tag)
		if err != nil {
			return nil, fmt.Errorf("failed to parse tags at %s.%s:\n\t * %w", path, field.Name, err)
		}
		fieldNativeName := field.Name
		fieldNativeExported := field.IsExported()

		// Extract the public field name (that's the content of `json:"XXX"` if we're deserializing JSON).
		// We'll use for deserialization and also for error messages, as we expect that the errors will
		// be readable by external users.
		publicFieldName := tags.PublicFieldName(options.renamingTagName)
		if publicFieldName == nil {
			publicFieldName = &fieldNativeName
		}

		hasDefault := tags.Default() != nil
		hasConstructionMethod := tags.MethodName() != nil

		if hasDefault && hasConstructionMethod {
			return nil, fmt.Errorf("struct %s contains a field \"%s\" that has both a `default` and a `orMethod` declaration. Please specify only one", path, fieldNativeName)
		}

		// By Go convention, a field with lower-case name or with a publicFieldName of "-" is private and
		// should not be parsed.
		isPublic := (*publicFieldName != "-") && fieldNativeExported
		if !isPublic && !initializationData.willPreinitialize {
			return nil, fmt.Errorf("struct %s contains a field \"%s\" that is not public, you should either make it public or specify an initializer with `Initializer` or `UnmarshalJSON`", path, fieldNativeName)
		}

		fieldPath := fmt.Sprint(path, ".", *publicFieldName)

		var fieldContentDeserializer reflectDeserializer
		fieldContentDeserializer, err = makeFieldDeserializerFromReflect(fieldPath, fieldType, options, &tags, selfContainer, initializationData.willPreinitialize)
		if err != nil {
			return nil, err
		}
		fieldDeserializer := func(outPtr *reflect.Value, inMap shared.Dict) error {
			// Note: maps are references, so there is no loss to passing a `map` instead of a `*map`.
			// Use the `fieldName` to access the field in the record.
			outReflect := outPtr.FieldByName(fieldNativeName)

			// Use the `publicFieldName` to access the field in the map.
			var fieldValue shared.Value
			if isPublic {
				// If the field is public, we can accept external data, if provided.
				var ok bool
				fieldValue, ok = inMap.Lookup(*publicFieldName)
				if !ok {
					fieldValue = nil
				}
			}
			err := fieldContentDeserializer(&outReflect, fieldValue)
			if err != nil {
				return err
			}

			// At this stage, the field has already been validated by using `Validator.Validate()`.
			// In future versions, we may wish to add support for further validation using tags.
			return nil
		}

		deserializers[field.Name] = fieldDeserializer
	}

	// True if this struct has a default value of {}.
	isZeroDefault := false
	if defaultSource := tags.Default(); defaultSource != nil {
		if *defaultSource == "{}" {
			isZeroDefault = true
		} else {
			return nil, fmt.Errorf("at %s, invalid `default` value. The only supported `default` value for structs is \"{}\", got: %s", path, *defaultSource)
		}
	}
	orMethod, err := makeOrMethodConstructor(tags, typ, container)
	if err != nil {
		return nil, fmt.Errorf("at %s, failed to setup `orMethod`\n\t * %w", path, err)
	}

	result := func(outPtr *reflect.Value, inValue shared.Value) (err error) {
		resultPtr := reflect.New(typ)
		result := resultPtr.Elem()

		// If possible, perform pre-initialization with default values.
		if initializer, ok := resultPtr.Interface().(validation.Initializer); ok {
			err = initializer.Initialize()
			wasPreInitialized = true
			if err != nil {
				err = fmt.Errorf("at %s, encountered an error while initializing optional fields:\n\t * %w", path, err)
				slog.Error("Internal error during deserialization", "error", err)
				return CustomDeserializerError{
					Wrapped:   err,
					Operation: "initializer",
					Structure: "struct",
				}
			}
		}

		// Don't forget to perform validation (unless we're returning an error).
		defer func() {
			if err != nil {
				// We're already returning an error, no need to insist.
				return
			}
			mightValidate := resultPtr.Interface()
			if validator, ok := mightValidate.(validation.Validator); ok {
				err = validator.Validate()
				if err != nil {
					// Validation error, abort struct construction.
					err = fmt.Errorf("deserialized value %s did not pass validation\n\t * %w", path, err)
					result = reflect.Zero(typ)
				}
			}
		}()
		switch {
		case inValue != nil:
			// We have all the data we need, proceed.
		case isZeroDefault || wasPreInitialized:
			inValue = internal.EmptyValue{}
		case orMethod != nil:
			constructed, err := (*orMethod)()
			if err != nil {
				err = fmt.Errorf("error in optional value at %s\n\t * %w", path, err)
				slog.Error("Internal error during deserialization", "error", err)
				return CustomDeserializerError{
					Wrapped:   err,
					Operation: "orMethod",
					Structure: "struct",
				}
			}
			reflected := reflect.ValueOf(constructed)
			outPtr.Set(reflected)
			return nil
		default:
			err = fmt.Errorf("missing object value at %s, expected %s", path, typeName(typ))
			return err
		}

		if initializationData.canUnmarshalSelf {
			resultPtrAny := resultPtr.Interface()
			err = options.unmarshaler.Unmarshal(inValue, &resultPtrAny)
			if err != nil {
				err = fmt.Errorf("at %s, expected to be able to parse a %s:\n\t * %w", path, typeName(typ), err)
				return err
			}
		} else {
			inMap, ok := inValue.AsDict()
			if !ok {
				err = fmt.Errorf("invalid value at %s, expected an object of type %s, got %s", path, typeName(typ), result.Type().Name())
				return err
			}

			// We may now deserialize fields.
			for _, fieldDeserializationData := range deserializers {
				err = fieldDeserializationData(&result, inMap)
				if err != nil {
					return err
				}
			}
		}

		outPtr.Set(result)
		return nil
	}
	return result, nil
}

// Construct a dynamically-typed deserializer for maps.
//
//   - `path` the human-readable path into the data structure, used for error-reporting;
//   - `typ` the dynamic type for the struct being compiled;
//   - `tags` the table of tags for this field.
//   - `wasPreinitialized` if this value was preinitialized, typically through `Initializer`
func makeMapDeserializerFromReflect(path string, typ reflect.Type, options staticOptions, tags *tagsPkg.Tags, container reflect.Value, wasPreInitialized bool) (reflectDeserializer, error) {
	if typ.Kind() != reflect.Map {
		panic(fmt.Sprintf("invalid call: %s is not a map", path))
	}
	if typ.Key().Kind() != reflect.String {
		return nil, fmt.Errorf("invalid map type at %s, only map[string]T can be converted into a deserializer", path)
	}

	// From this point, we know that it's a `map[string]T` for some `T`.
	selfContainer := reflect.New(typ)

	initializationMetadata, err := initializationData(path, typ, options)
	if err != nil {
		return nil, err
	}

	subPath := fmt.Sprintf("%s[]", path)
	subTags := tagsPkg.Empty()
	subTyp := typ.Elem()
	contentDeserializer, err := makeFieldDeserializerFromReflect(subPath, subTyp, options, &subTags, selfContainer, initializationMetadata.willPreinitialize)
	if err != nil {
		return nil, err
	}

	// True if this map has a default value of {}.
	isZeroDefault := false
	if defaultSource := tags.Default(); defaultSource != nil {
		if *defaultSource == "{}" {
			isZeroDefault = true
		} else {
			return nil, fmt.Errorf("at %s, invalid `default` value. The only supported `default` value for maps is \"{}\", got: %s", path, *defaultSource)
		}
	}
	orMethod, err := makeOrMethodConstructor(tags, typ, container)
	if err != nil {
		return nil, fmt.Errorf("at %s, failed to setup `orMethod`\n\t * %w", path, err)
	}

	result := func(outPtr *reflect.Value, inValue shared.Value) (err error) {
		result := reflect.MakeMap(typ)

		// If possible, perform pre-initialization with default values.
		if initializer, ok := result.Interface().(validation.Initializer); ok {
			err = initializer.Initialize()
			wasPreInitialized = true
			if err != nil {
				err = fmt.Errorf("at %s, encountered an error while initializing optional object:\n\t * %w", path, err)
				slog.Error("Internal error during deserialization", "error", err)
				return CustomDeserializerError{
					Wrapped:   err,
					Operation: "initialize",
					Structure: "map",
				}
			}
		}

		// Don't forget to perform validation (unless we're returning an error).
		defer func() {
			if err != nil {
				// We're already returning an error, no need to insist.
				return
			}
			mightValidate := result.Interface()
			if validator, ok := mightValidate.(validation.Validator); ok {
				err = validator.Validate()
				if err != nil {
					// Validation error, abort struct construction.
					err = fmt.Errorf("deserialized value %s did not pass validation\n\t * %w", path, err)
					result = reflect.Zero(typ)
				}
			}
		}()
		print("TEST")
		switch {
		case inValue != nil:
			// We have all the data we need, proceed.
		case isZeroDefault || wasPreInitialized:
			inValue = internal.EmptyDict{}.AsValue()
		case orMethod != nil:
			constructed, err := (*orMethod)()
			if err != nil {
				err = fmt.Errorf("error in optional object at %s\n\t * %w", path, err)
				slog.Error("Internal error during deserialization", "error", err)
				return CustomDeserializerError{
					Wrapped:   err,
					Operation: "orMethod",
					Structure: "map",
				}
			}
			reflected := reflect.ValueOf(constructed)
			outPtr.Set(reflected)
			return nil
		default:
			err = fmt.Errorf("missing object value at %s, expected %s", path, typeName(typ))
			return err
		}

		if initializationMetadata.canUnmarshalSelf {
			// Our struct supports `Unmarshaler`. This means that:
			//
			// - it enforces its own invariants (we do not perform pre-initialization or in-depth validation);
			// - from the point of view of the outside world, this MUST be a string.
			resultPtrAny := result.Interface()
			err = options.unmarshaler.Unmarshal(inValue.Interface(), &resultPtrAny)
			if err != nil {
				err = fmt.Errorf("invalid string at %s, expected to be able to parse a %s:\n\t * %w", path, typeName(typ), err)
				return err
			}
		} else {
			inMap, ok := inValue.AsDict()
			if !ok {
				err = fmt.Errorf("invalid value at %s, expected an object of type %s, got %v", path, typeName(typ), inValue.Interface())
				return err
			}

			// We may now deserialize keys and values.
			keys := inMap.Keys()
			for _, k := range keys {
				subInValue, ok := inMap.Lookup(k)
				if !ok {
					slog.Error("Internal error while ranging over map: missing value", "path", path, "key", k)
					// Hobble on.
					continue
				}

				reflectedContent := reflect.New(subTyp).Elem()
				err = contentDeserializer(&reflectedContent, subInValue)
				if err != nil {
					return err
				}
				result.SetMapIndex(reflect.ValueOf(k), reflectedContent)
			}
		}

		outPtr.Set(result)
		return nil
	}
	return result, nil
}

// Construct a dynamically-typed deserializer for slices.
//
//   - `path` the human-readable path into the data structure, used for error-reporting;
//   - `typ` the dynamic type for the slice being compiled;
//   - `tagName` the name of tags to use for field renamings, e.g. `query`;
//   - `tags` the table of tags for this field.
func makeSliceDeserializer(fieldPath string, fieldType reflect.Type, options staticOptions, tags *tagsPkg.Tags, container reflect.Value, wasPreinitialized bool) (reflectDeserializer, error) {
	arrayPath := fmt.Sprint(fieldPath, "[]")
	isEmptyDefault := false
	if defaultSource := tags.Default(); defaultSource != nil {
		if *defaultSource == "[]" {
			isEmptyDefault = true
		} else {
			return nil, fmt.Errorf("at %s, invalid `default` value. The only supported `default` value for arrays or slices is \"[]\", got: %s", fieldPath, *defaultSource)
		}
	}
	orMethod, err := makeOrMethodConstructor(tags, fieldType, container)
	if err != nil {
		return nil, fmt.Errorf("at %s, failed to setup `orMethod`\n\t * %w", fieldPath, err)
	}

	// Early check that we're not misusing Validator.
	_, err = canInterface(fieldType, validatorInterface)
	if err != nil {
		return nil, err
	}

	subTags := tagsPkg.Empty()
	subContainer := reflect.New(fieldType).Elem()

	// Prepare a deserializer for elements in this slice.
	childPreinitialized := tags.IsPreinitialized()
	elementDeserializer, err := makeFieldDeserializerFromReflect(arrayPath, fieldType.Elem(), options, &subTags, subContainer, childPreinitialized)
	if err != nil {
		return nil, fmt.Errorf("failed to generate a deserializer for %s\n\t * %w", fieldPath, err)
	}
	result := func(outPtr *reflect.Value, inValue shared.Value) (err error) {
		var reflectedResult reflect.Value

		// Don't forget to perform validation (unless we're returning an error).
		defer func() {
			if err != nil {
				// We're already returning an error, no need to insist.
				return
			}
			mightValidate := outPtr.Interface()
			if validator, ok := mightValidate.(validation.Validator); ok {
				err = validator.Validate()
				if err != nil {
					// Validation error, abort struct construction.
					err = fmt.Errorf("deserialized value %s did not pass validation\n\t * %w", fieldPath, err)
					outPtr.Set(reflect.Zero(fieldType))
				}
			}
		}()

		// Move into array
		var input []shared.Value
		switch {
		case inValue != nil:
			// Simply deserialize.
			var ok bool
			if input, ok = inValue.AsSlice(); !ok {
				return fmt.Errorf("error while deserializing %s[]: expected an array", fieldType)
			}
		case isEmptyDefault:
			// Nothing to deserialize, but we are allowed to default to an empty array.
			input = make([]shared.Value, 0)
		case orMethod != nil:
			// Nothing to deserialize, but we know how to build a default value.
			orMethodResult, err := (*orMethod)()
			if err != nil {
				return fmt.Errorf("error in optional value at %s\n\t * %w", fieldPath, err)
			}
			reflectedOrMethodSlice := reflect.ValueOf(orMethodResult)
			result := reflect.MakeSlice(fieldType, 0, reflectedOrMethodSlice.Len())
			result = reflect.AppendSlice(result, reflectedOrMethodSlice)
			outPtr.Set(result)
			return nil
		case wasPreinitialized:
			// No value? That's ok, we got a value from preinitialization.
		default:
			return fmt.Errorf("missing value at %s, expected an array of %s", arrayPath, fieldPath)
		}

		reflectedResult = reflect.MakeSlice(fieldType, len(input), len(input))

		// Recurse into entries.
		for i, inAtIndex := range input {
			outAtIndex := reflectedResult.Index(i)
			err := elementDeserializer(&outAtIndex, inAtIndex)
			if err != nil {
				return fmt.Errorf("error while deserializing %s[%d]:\n\t * %w", fieldPath, i, err)
			}
			reflect.Append(reflectedResult, outAtIndex)
		}
		outPtr.Set(reflectedResult)
		return nil
	}
	return result, nil
}

// Construct a dynamically-typed deserializer for pointers.
//
//   - `path` the human-readable path into the data structure, used for error-reporting;
//   - `typ` the dynamic type for the pointer being compiled;
//   - `tagName` the name of tags to use for field renamings, e.g. `query`;
//   - `tags` the table of tags for this field.
func makePointerDeserializer(fieldPath string, fieldType reflect.Type, options staticOptions, tags *tagsPkg.Tags, container reflect.Value, wasPreinitialized bool) (reflectDeserializer, error) {
	ptrPath := fmt.Sprint(fieldPath, "*")
	elemType := fieldType.Elem()
	subTags := tagsPkg.Empty()
	subContainer := reflect.New(fieldType).Elem()
	childPreinitialized := tags.IsPreinitialized()
	elementDeserializer, err := makeFieldDeserializerFromReflect(ptrPath, fieldType.Elem(), options, &subTags, subContainer, childPreinitialized)
	if err != nil {
		return nil, fmt.Errorf("failed to generate a deserializer for %s\n\t * %w", fieldPath, err)
	}

	// True if we support `nil` as default value.
	isNilDefault := false
	if defaultSource := tags.Default(); defaultSource != nil {
		if *defaultSource == "nil" {
			isNilDefault = true
		} else {
			return nil, fmt.Errorf("at %s, invalid `default` value. The only supported `default` value for pointers is \"nil\", got: %s", fieldPath, *defaultSource)
		}
	}
	orMethod, err := makeOrMethodConstructor(tags, fieldType, container)
	if err != nil {
		return nil, fmt.Errorf("at %s, failed to setup `orMethod`\n\t * %w", fieldPath, err)
	}

	result := func(outPtr *reflect.Value, inValue shared.Value) (err error) {
		switch {
		case inValue != nil:
			// We have all the data we need, proced.
		case wasPreinitialized:
			// No value? That's ok, we got a value from preinitialization.
			return nil
		case isNilDefault:
			// No value? That's ok for a pointer.
			*outPtr = reflect.ValueOf(nil)
			return nil
		case orMethod != nil:
			result, err := (*orMethod)()
			if err != nil {
				err = fmt.Errorf("error in optional value at %s\n\t * %w", fieldPath, err)
				slog.Error("Internal error during deserialization", "error", err)
				return CustomDeserializerError{
					Wrapped:   err,
					Operation: "orMethod",
					Structure: "ptr",
				}
			}
			outPtr.Set(reflect.ValueOf(result))
			return nil
		}

		// Move into ptr
		reflectedPtrResult := reflect.New(elemType)
		reflectedResult := reflectedPtrResult.Elem()
		err = elementDeserializer(&reflectedResult, inValue)
		if err != nil {
			return err //nolint:wrapcheck
		}

		// Note: We do not perform validation here as validation has already happened
		// when constructing the value we're pointing at.
		outPtr.Set(reflectedPtrResult)
		return nil
	}
	return result, nil
}

// Construct a dynamically-typed deserializer for a flat field (string, int, etc.).
//
//   - `path` the human-readable path into the data structure, used for error-reporting;
//   - `typ` the dynamic type for the field being compiled;
//   - `tagName` the name of tags to use for field renamings, e.g. `query`;
//   - `tags` the table of tags for this field.
func makeFlatFieldDeserializer(fieldPath string, fieldType reflect.Type, options staticOptions, tags *tagsPkg.Tags, container reflect.Value, wasPreinitialized bool) (reflectDeserializer, error) {
	typeName := typeName(fieldType)

	// A parser in case we receive our data as a string.
	parser := shared.LookupParser(fieldType)

	// An unmarshaler in case we receive our data as... something else.
	var unmarshaler *func(any) (any, error)
	if options.unmarshaler.ShouldUnmarshal(fieldType) {
		u := func(source any) (any, error) {
			result := reflect.New(fieldType).Interface()
			err := options.unmarshaler.Unmarshal(source, &result)
			if err != nil {
				err = fmt.Errorf("invalid data at, expected to be able to parse a %s:\n\t * %w", typeName, err)
				return nil, err
			}
			return result, nil
		}
		unmarshaler = &u
	}
	// Early check that we're not misusing Validator.
	_, err := canInterface(fieldType, validatorInterface)
	if err != nil {
		return nil, err
	}

	// If a `default` tag is provided, the parsed default value.
	var defaultValue any
	if defaultSource := tags.Default(); defaultSource != nil {
		// Attempt to generate a default value.
		if parser == nil {
			return nil, fmt.Errorf("cannot specify a default value at %s for type %s as we don't have a parser for such values", fieldPath, fieldType)
		}
		var err error
		defaultValue, err = (*parser)(*defaultSource)
		if err != nil {
			return nil, fmt.Errorf("cannot parse default value at %s\n\t * %w", fieldPath, err)
		}
	}

	// If a `orMethod` tag is provided, a closure to call this method.
	orMethod, err := makeOrMethodConstructor(tags, fieldType, container)
	if err != nil {
		return nil, fmt.Errorf("at %s, failed to setup `orMethod`\n\t * %w", fieldPath, err)
	}
	result := func(outPtr *reflect.Value, inValue shared.Value) (err error) {
		var reflectedInput reflect.Value

		// Don't forget to perform validation (unless we're returning an error).
		defer func() {
			if err != nil {
				// We're already returning an error, no need to insist.
				return
			}
			mightValidate := outPtr.Interface()
			if validator, ok := mightValidate.(validation.Validator); ok {
				err = validator.Validate()
				if err != nil {
					// Validation error, abort struct construction.
					err = fmt.Errorf("deserialized value %s did not pass validation\n\t * %w", fieldPath, err)
					outPtr.Set(reflect.Zero(fieldType))
				}
			}
		}()

		var input any
		switch {
		case inValue != nil:
			// We have all the data we need, proceed.
			input = inValue.Interface()
		case wasPreinitialized:
			input = outPtr.Interface()
		case defaultValue != nil:
			input = defaultValue
		case orMethod != nil:
			constructed, err := (*orMethod)()
			if err != nil {
				err = fmt.Errorf("error in optional value at %s\n\t * %w", fieldPath, err)
				slog.Error("Internal error during deserialization", "error", err)
				return CustomDeserializerError{
					Wrapped:   err,
					Operation: "orMethod",
					Structure: "field",
				}
			}
			input = constructed
		default:
			return fmt.Errorf("missing value at %s, expected %s", fieldPath, typeName)
		}

		// Type check: can our value convert to the expected type?
		reflectedInput = reflect.ValueOf(input)
		ok := reflectedInput.CanConvert(fieldType)
		if !ok {
			// The input cannot be converted?
			//
			// Perhaps we can fix it.
			recovered := false
			var parsed any
			if parser != nil {
				if inputString, ok := input.(string); ok {
					// The input is represented as a string, but we're not looking for a
					// string. This can happen e.g. for queries or paths, for which
					// everything is a string, or for json bodies, in case of client error.
					//
					// Regardless, let's try and convert.
					parsed, err = (*parser)(inputString)
					if err == nil {
						recovered = true
					}
				}
			}
			if !recovered && unmarshaler != nil {
				parsed, err = (*unmarshaler)(input)
				if err == nil {
					recovered = true
				}
			}
			if recovered {
				input = parsed
			} else {
				return fmt.Errorf("invalid value at %s, expected %s, got %s", fieldPath, typeName, reflectedInput.Type().Name())
			}
			reflectedInput = reflect.ValueOf(input)
		}
		reflectedInput = reflectedInput.Convert(fieldType)
		outPtr.Set(reflectedInput)
		return nil
	}
	return result, nil
}

// Construct a dynamically-typed deserializer for any field.
//
//   - `path` the human-readable path into the data structure, used for error-reporting;
//   - `typ` the dynamic type for the field being compiled;
//   - `tagName` the name of tags to use for field renamings, e.g. `query`;
//   - `tags` the table of tags for this field.
func makeFieldDeserializerFromReflect(fieldPath string, fieldType reflect.Type, options staticOptions, tags *tagsPkg.Tags, container reflect.Value, wasPreinitialized bool) (reflectDeserializer, error) {
	var result reflectDeserializer
	var err error
	switch fieldType.Kind() {
	case reflect.Pointer:
		result, err = makePointerDeserializer(fieldPath, fieldType, options, tags, container, wasPreinitialized)
	case reflect.Array:
		fallthrough
	case reflect.Slice:
		if options.allowNested {
			result, err = makeSliceDeserializer(fieldPath, fieldType, options, tags, container, wasPreinitialized)
		} else {
			return nil, fmt.Errorf("this type of extractor does not support arrays/slices")
		}
	case reflect.Struct:
		if options.allowNested {
			result, err = makeStructDeserializerFromReflect(fieldPath, fieldType, options, tags, container, wasPreinitialized)
		} else {
			return nil, fmt.Errorf("this type of extractor does not support nested structs")
		}
	case reflect.Map:
		if options.allowNested {
			result, err = makeMapDeserializerFromReflect(fieldPath, fieldType, options, tags, container, wasPreinitialized)
		} else {
			return nil, fmt.Errorf("this type of extractor does not support nested maps")
		}
	default:
		// If it's not a struct, an array, a slice or a pointer, well, it's probably something flat.
		//
		// We'll let `makeFlatFieldDeserializer` detect whether we can generate a deserializer for it.
		result, err = makeFlatFieldDeserializer(fieldPath, fieldType, options, tags, container, wasPreinitialized)
	}
	if err != nil {
		return nil, fmt.Errorf("could not generate a deserializer for %s with type %s:\n\t * %w", fieldPath, typeName(fieldType), err)
	}
	return result, nil
}

// Return a (mostly) human-readable type name for a Go type.
//
// This type name is used for user error messages.
func typeName(typ reflect.Type) string {
	fullName := typ.Name()
	pkgName := fmt.Sprint(typ.PkgPath(), ".")
	return strings.ReplaceAll(fullName, pkgName, "")
}

// A custom constructor provided with tag `orMethod`.
type orMethodConstructor func() (any, error)

func makeOrMethodConstructor(tags *tagsPkg.Tags, fieldType reflect.Type, container reflect.Value) (*orMethodConstructor, error) {
	var defaultMethodConstructor *orMethodConstructor
	if defaultMethodConstructorName := tags.MethodName(); defaultMethodConstructorName != nil {
		method := container.MethodByName(*defaultMethodConstructorName)
		if method.IsValid() {
			typ := method.Type()
			switch {
			case typ.NumIn() != 0:
				return nil, fmt.Errorf("the method provided with `orMethod` MUST take no argument but takes %d arguments", typ.NumIn())
			case typ.NumOut() != 2: //nolint:gomnd
				return nil, fmt.Errorf("the method provided with `orMethod` MUST return (%s, error) but it returns %d value(s)", fieldType.Name(), typ.NumOut())
			case !typ.Out(0).ConvertibleTo(fieldType):
				return nil, fmt.Errorf("the method provided with `orMethod` MUST return (%s, error) but it returns (%s, _) which is not convertible to `%s`", fieldType.Name(), typ.Out(0).Name(), fieldType.Name())
			case !typ.Out(1).ConvertibleTo(errorInterface):
				return nil, fmt.Errorf("the method provided with `orMethod` MUST return (%s, error) but it returns (_, %s) which is not convertible to `error`", fieldType.Name(), typ.Out(1).Name())
			}
			args := make([]reflect.Value, 0)
			var methodConstructor orMethodConstructor = func() (any, error) {
				out := method.Call(args)
				result := out[0].Interface() // We have just checked that it MUST be convertible to `any`.
				var err error
				err, ok := out[1].Interface().(error) // We have just checked that it MUST be convertible to `error`.
				if !ok {
					// Conversion fails if `error` is `nil`.
					return result, nil
				}
				return result, err
			}
			defaultMethodConstructor = &methodConstructor
		} else {
			return nil, fmt.Errorf("method %s provided with `orMethod` doesn't seem to exist - note that the method must be public", *defaultMethodConstructorName)
		}
	}
	return defaultMethodConstructor, nil
}

// Check that a type implements an interface *on pointers*.
func canInterface(typ reflect.Type, interfaceType reflect.Type) (bool, error) {
	ptrTyp := reflect.PointerTo(typ)
	if typ.Implements(interfaceType) {
		return false, fmt.Errorf("type %s implements %s - it should be implemented by pointer type *%s instead", typ, interfaceType, typ)
	}
	if ptrTyp.ConvertibleTo(interfaceType) {
		return true, nil
	}
	return false, nil
}

// Some metadata on initialization for a type.
type initializationMetadata struct {
	canInitializeSelf bool
	canUnmarshalSelf  bool
	willPreinitialize bool
}

func initializationData(path string, typ reflect.Type, options staticOptions) (initializationMetadata, error) {
	// If this structure supports self-initialization or custom unmarshaling, we don't need (or use)
	// default fields and `orMethod` constructors.
	canInitializeSelf, err := canInterface(typ, initializerInterface)
	if err != nil {
		return initializationMetadata{}, err
	}

	canUnmarshalSelf := options.unmarshaler.ShouldUnmarshal(typ)
	if canInitializeSelf && canUnmarshalSelf {
		slog.Warn("Type supports both Initializer and Unmarshaler, defaulting to Unmarshaler", "path", path, "type", typ)
		canInitializeSelf = false
	}
	willPreinitialize := canInitializeSelf || canUnmarshalSelf

	// Early check that we're not mis-using `Validator`.
	_, err = canInterface(typ, validatorInterface)
	if err != nil {
		return initializationMetadata{}, err
	}

	return initializationMetadata{
		canInitializeSelf: canInitializeSelf,
		canUnmarshalSelf:  canUnmarshalSelf,
		willPreinitialize: willPreinitialize,
	}, nil
}
