//nolint:exhaustruct
package deserialize_test

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/pasqal-io/godasse/deserialize"
	jsonPkg "github.com/pasqal-io/godasse/deserialize/json"
	"github.com/pasqal-io/godasse/deserialize/kvlist"
	"github.com/pasqal-io/godasse/deserialize/shared"
	"github.com/pasqal-io/godasse/validation"
	"gotest.tools/v3/assert"
)

type SimpleStruct struct {
	SomeString string
}

type EmptyStruct struct{}

type PrimitiveTypesStruct struct {
	SomeBool    bool
	SomeString  string
	SomeFloat32 float32
	SomeFloat64 float64
	SomeInt     int
	SomeInt8    int8
	SomeInt16   int16
	SomeInt32   int32
	SomeInt64   int64
	SomeUint8   uint8
	SomeUint16  uint16
	SomeUint32  uint32
	SomeUint64  uint64
}

type SimpleArrayStruct struct {
	SomeSlice []string
	SomeArray [3]string
}

type ArrayOfStructs struct {
	SomeArray []SimpleStruct
}

type Pair[T any, U any] struct {
	Left  T `json:"left"`
	Right U `json:"right"`
}

type Array[T any] struct {
	Data []T
}

type ValidatedStruct struct {
	SomeEmail string
}

// Validate implements validation.Validator.
func (s *ValidatedStruct) Validate() error {
	if strings.Contains(s.SomeEmail, "@") {
		return nil
	}
	return errors.New("Invalid email")
}

var _ validation.Validator = &ValidatedStruct{} // Type assertion.

func twoWaysGeneric[Input any, Output any](t *testing.T, sample Input) (*Output, error) {
	deserializer, err := deserialize.MakeMapDeserializer[Output](deserialize.Options{
		Unmarshaler: jsonPkg.Driver,
		MainTagName: "json",
	})
	if err != nil {
		t.Error(err)
		return nil, err
	}

	buf, err := json.Marshal(sample)
	if err != nil {
		t.Error(err)
		return nil, err //nolint:wrapcheck
	}
	dict := make(jsonPkg.JSON)
	err = json.Unmarshal(buf, &dict)
	if err != nil {
		t.Error(err)
		return nil, err //nolint:wrapcheck
	}
	return deserializer.DeserializeDict(dict) //nolint:wrapcheck
}
func twoWays[T any](t *testing.T, sample T) (*T, error) {
	return twoWaysGeneric[T, T](t, sample)
}

func twoWaysListGeneric[Input any, Output any](t *testing.T, samples []Input) ([]Output, error) {
	deserializer, err := deserialize.MakeMapDeserializer[Output](deserialize.Options{
		Unmarshaler: jsonPkg.Driver,
		MainTagName: "json",
	})
	if err != nil {
		t.Error(err)
		return nil, err
	}

	buf, err := json.Marshal(samples)
	if err != nil {
		t.Error(err)
		return nil, err //nolint:wrapcheck
	}
	unmarshalList := []any{}
	err = json.Unmarshal(buf, &unmarshalList)
	if err != nil {
		t.Error(err)
		return nil, err //nolint:wrapcheck
	}
	list := []shared.Value{}
	for _, entry := range unmarshalList {
		list = append(list, jsonPkg.Driver().WrapValue(entry))
	}
	return deserializer.DeserializeList(list) //nolint:wrapcheck
}
func twoWaysList[T any](t *testing.T, samples []T) ([]T, error) {
	return twoWaysListGeneric[T, T](t, samples)
}

func TestMissingOptions(t *testing.T) {
	var err error

	_, err = deserialize.MakeMapDeserializer[SimpleStruct](deserialize.Options{})
	assert.ErrorContains(t, err, "missing option MainTagName")

	_, err = deserialize.MakeKVListDeserializer[SimpleStruct](deserialize.Options{})
	assert.ErrorContains(t, err, "missing option MainTagName")

	_, err = deserialize.MakeMapDeserializer[SimpleStruct](deserialize.Options{})
	assert.ErrorContains(t, err, "missing option MainTagName")

	_, err = deserialize.MakeMapDeserializer[SimpleStruct](deserialize.Options{
		MainTagName: "foo",
	})
	assert.ErrorContains(t, err, "please specify an unmarshaler")

	_, err = deserialize.MakeKVListDeserializer[SimpleStruct](deserialize.Options{
		MainTagName: "foo",
	})
	assert.ErrorContains(t, err, "please specify an unmarshaler")

	_, err = deserialize.MakeMapDeserializer[SimpleStruct](deserialize.Options{
		MainTagName: "foo",
	})
	assert.ErrorContains(t, err, "please specify an unmarshaler")
}

func TestDeserializeMapBufAndString(t *testing.T) {
	deserializer, err := deserialize.MakeMapDeserializer[PrimitiveTypesStruct](deserialize.JSONOptions(""))
	assert.NilError(t, err)
	sample := PrimitiveTypesStruct{
		SomeBool:    true,
		SomeString:  "Some String",
		SomeFloat32: 32.0,
		SomeFloat64: 64.0,
		SomeInt:     -42,
		SomeInt8:    -8,
		SomeInt16:   -16,
		SomeInt32:   -32,
		SomeInt64:   -64,
		SomeUint8:   8,
		SomeUint16:  16,
		SomeUint32:  32,
		SomeUint64:  64,
	}
	var buf []byte
	buf, err = json.Marshal(sample)
	assert.NilError(t, err)

	var result *PrimitiveTypesStruct
	result, err = deserializer.DeserializeBytes(buf)
	assert.NilError(t, err)

	assert.Equal(t, *result, sample, "We should have succeeded when deserializing from bytes")

	result, err = deserializer.DeserializeString(string(buf))
	assert.NilError(t, err)
	assert.Equal(t, *result, sample, "We should have succeeded when deserializing from string")
}

func TestDeserializeSimpleStruct(t *testing.T) {
	before := SimpleStruct{
		SomeString: "THIS IS A TEST",
	}
	after, err := twoWays[SimpleStruct](t, before)
	assert.NilError(t, err)
	assert.Equal(t, *after, before, "We should have recovered the same struct")
}

func TestDeserializeSimpleStructList(t *testing.T) {
	before := []SimpleStruct{
		{
			SomeString: "THIS IS A TEST",
		},
	}
	after, err := twoWaysList[SimpleStruct](t, before)
	assert.NilError(t, err)
	assert.DeepEqual(t, after, before)
}

// Test with all primitive types.
func TestDeserializeSimpleTypes(t *testing.T) {
	before := PrimitiveTypesStruct{
		SomeBool:    true,
		SomeString:  "text",
		SomeFloat32: -1.0,
		SomeFloat64: -2.0,
		SomeInt:     -1,
		SomeInt8:    -2,
		SomeInt16:   -3,
		SomeInt32:   -4,
		SomeInt64:   -5,
		SomeUint8:   6,
		SomeUint16:  7,
		SomeUint32:  8,
		SomeUint64:  9,
	}
	after, err := twoWays[PrimitiveTypesStruct](t, before)
	assert.NilError(t, err)
	assert.Equal(t, *after, before, "We should have recovered the same struct")
}

// If a field is missing, we should fail.
func TestDeserializeMissingField(t *testing.T) {
	before := SimpleStruct{
		SomeString: "text",
	}
	_, err := twoWaysGeneric[SimpleStruct, PrimitiveTypesStruct](t, before)
	assert.ErrorContains(t, err, "missing value at PrimitiveTypesStruct", "We should have recovered the same struct")
}

// Test with generics.
func TestGenerics(t *testing.T) {
	before := Pair[PrimitiveTypesStruct, SimpleStruct]{
		Left: PrimitiveTypesStruct{
			SomeBool:    true,
			SomeString:  "text",
			SomeFloat32: -1.0,
			SomeFloat64: -2.0,
			SomeInt:     -1,
			SomeInt8:    -2,
			SomeInt16:   -3,
			SomeInt32:   -4,
			SomeInt64:   -5,
			SomeUint8:   6,
			SomeUint16:  7,
			SomeUint32:  8,
			SomeUint64:  9,
		},
		Right: SimpleStruct{
			SomeString: "More text",
		},
	}
	after, err := twoWays[Pair[PrimitiveTypesStruct, SimpleStruct]](t, before)
	assert.NilError(t, err)
	assert.Equal(t, *after, before, "We should have recovered the same struct")
}

func TestDeserializeDeepMissingField(t *testing.T) {
	before := Pair[int, SimpleStruct]{
		Left: 123,
		Right: SimpleStruct{
			SomeString: "text",
		},
	}
	_, err := twoWaysGeneric[Pair[int, SimpleStruct], Pair[int, PrimitiveTypesStruct]](t, before)
	assert.ErrorContains(t, err, "missing value at Pair[int,PrimitiveTypesStruct].right", "We should have detected the missing value")
}

func TestDeserializeSimpleArray(t *testing.T) {
	array := [3]string{"one", "ttwo", "three"}
	slice := []string{"four", "fife", "six"}
	before := SimpleArrayStruct{
		SomeSlice: slice,
		SomeArray: array,
	}
	after, err := twoWays(t, before)
	if err != nil {
		t.Error(err)
	}
	assert.DeepEqual(t, after, &before)
}

func TestDeserializeArrayOfStruct(t *testing.T) {
	array := []SimpleStruct{{"one"}, {"two"}, {"three"}}
	before := ArrayOfStructs{
		SomeArray: array,
	}
	after, err := twoWays(t, before)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, len(after.SomeArray), len(before.SomeArray), "We should have recovered the same struct")
	for i := 0; i < len(before.SomeArray); i++ {
		assert.Equal(t, after.SomeArray[i], before.SomeArray[i], "We should have recovered the same struct")
	}
}

func TestValidationSuccess(t *testing.T) {
	before := ValidatedStruct{
		SomeEmail: "someone@example.com",
	}
	after, err := twoWays(t, before)
	assert.NilError(t, err)
	assert.Equal(t, *after, before, "We should have recovered the same struct")
}

func TestValidationFailureField(t *testing.T) {
	before := ValidatedStruct{
		SomeEmail: "someone+example.com",
	}
	_, err := twoWays(t, before)
	assert.ErrorContains(t, err, "ValidatedStruct")
	assert.ErrorContains(t, err, "Invalid email")
}

func TestValidationFailureFieldField(t *testing.T) {
	before := Pair[int, ValidatedStruct]{
		Left: 0,
		Right: ValidatedStruct{
			SomeEmail: "someone+example.com",
		},
	}
	_, err := twoWays(t, before)
	assert.ErrorContains(t, err, ".right")
	assert.ErrorContains(t, err, "Invalid email")
}

func TestValidationFailureArray(t *testing.T) {
	array := []ValidatedStruct{{
		SomeEmail: "someone+example.com",
	}}
	before := Array[ValidatedStruct]{
		Data: array,
	}
	_, err := twoWays(t, before)
	assert.ErrorContains(t, err, "Data[0]")
	assert.ErrorContains(t, err, "Invalid email")
}

func TestKVListSimple(t *testing.T) {
	options := deserialize.QueryOptions("") //nolint:exhaustruct
	deserializer, err := deserialize.MakeKVListDeserializer[PrimitiveTypesStruct](options)
	assert.NilError(t, err)

	var entry kvlist.KVList = make(kvlist.KVList)
	entry["SomeBool"] = []string{"true"}
	entry["SomeString"] = []string{"blue"}
	entry["SomeFloat32"] = []string{"3.14"}
	entry["SomeFloat64"] = []string{"1.69"}
	entry["SomeInt"] = []string{"-1"}
	entry["SomeInt8"] = []string{"-8"}
	entry["SomeInt16"] = []string{"-16"}
	entry["SomeInt32"] = []string{"-32"}
	entry["SomeInt64"] = []string{"-64"}
	entry["SomeUint8"] = []string{"8"}
	entry["SomeUint16"] = []string{"16"}
	entry["SomeUint32"] = []string{"32"}
	entry["SomeUint64"] = []string{"64"}

	sample := PrimitiveTypesStruct{
		SomeBool:    true,
		SomeString:  "blue",
		SomeFloat32: 3.14,
		SomeFloat64: 1.69,
		SomeInt:     -1,
		SomeInt8:    -8,
		SomeInt16:   -16,
		SomeInt32:   -32,
		SomeInt64:   -64,
		SomeUint8:   8,
		SomeUint16:  16,
		SomeUint32:  32,
		SomeUint64:  64,
	}

	deserialized, err := deserializer.DeserializeKVList(entry)
	assert.NilError(t, err)
	assert.Equal(t, *deserialized, sample, "We should have extracted the expected value")

}

// Test that if we place a string instead of a primitive type, this string
// will be parsed.
func TestConversionsSuccess(t *testing.T) {
	type StringAsPrimitiveTypesStruct struct {
		SomeBool    string
		SomeString  string
		SomeFloat32 string
		SomeFloat64 string
		SomeInt     string
		SomeInt8    string
		SomeInt16   string
		SomeInt32   string
		SomeInt64   string
		SomeUint8   string
		SomeUint16  string
		SomeUint32  string
		SomeUint64  string
	}
	sample := StringAsPrimitiveTypesStruct{
		SomeBool:    "true",
		SomeString:  "text",
		SomeFloat32: "-1.0",
		SomeFloat64: "-2.0",
		SomeInt:     "-1",
		SomeInt8:    "-2",
		SomeInt16:   "-3",
		SomeInt32:   "-4",
		SomeInt64:   "-5",
		SomeUint8:   "6",
		SomeUint16:  "7",
		SomeUint32:  "8",
		SomeUint64:  "9",
	}
	expected := PrimitiveTypesStruct{
		SomeBool:    true,
		SomeString:  "text",
		SomeFloat32: -1.0,
		SomeFloat64: -2.0,
		SomeInt:     -1,
		SomeInt8:    -2,
		SomeInt16:   -3,
		SomeInt32:   -4,
		SomeInt64:   -5,
		SomeUint8:   6,
		SomeUint16:  7,
		SomeUint32:  8,
		SomeUint64:  9,
	}
	result, err := twoWaysGeneric[StringAsPrimitiveTypesStruct, PrimitiveTypesStruct](t, sample)
	assert.NilError(t, err)
	assert.Equal(t, *result, expected, "Deserialization should have parsed strings")
}

// Test that if we place a string instead of a primitive type, this string
// will be parsed.
func TestConversionsFailure(t *testing.T) {
	type StringAsPrimitiveTypesStruct struct {
		SomeBool    string
		SomeFloat32 string
		SomeFloat64 string
		SomeInt     string
		SomeInt8    string
		SomeInt16   string
		SomeInt32   string
		SomeInt64   string
		SomeUint8   string
		SomeUint16  string
		SomeUint32  string
		SomeUint64  string
		SomeString  string
	}
	sample := StringAsPrimitiveTypesStruct{
		SomeBool:    "blue",
		SomeFloat32: "blue",
		SomeFloat64: "blue",
		SomeInt:     "blue",
		SomeInt8:    "blue",
		SomeInt16:   "blue",
		SomeInt32:   "blue",
		SomeInt64:   "blue",
		SomeUint8:   "blue",
		SomeUint16:  "blue",
		SomeUint32:  "blue",
		SomeUint64:  "blue",
		SomeString:  "string",
	}
	_, err := twoWaysGeneric[StringAsPrimitiveTypesStruct, PrimitiveTypesStruct](t, sample)
	assert.ErrorContains(t, err, "invalid value at PrimitiveTypesStruct", "We should detect syntax errors in default values during setup.")
}

// Test that default values are parsed.
func TestPrimitiveDefaultValues(t *testing.T) {
	type PrimitiveTypesStructWithDefault struct {
		SomeBool    bool    `default:"true"`
		SomeString  string  `default:"text"`
		SomeFloat32 float32 `default:"-1.0"`
		SomeFloat64 float64 `default:"-2.0"`
		SomeInt     int     `default:"-1"`
		SomeInt8    int8    `default:"-2"`
		SomeInt16   int16   `default:"-3"`
		SomeInt32   int32   `default:"-4"`
		SomeInt64   int64   `default:"-5"`
		SomeUint8   uint8   `default:"6"`
		SomeUint16  uint16  `default:"7"`
		SomeUint32  uint32  `default:"8"`
		SomeUint64  uint64  `default:"9"`
	}
	sample := EmptyStruct{}
	expected := PrimitiveTypesStructWithDefault{
		SomeBool:    true,
		SomeString:  "text",
		SomeFloat32: -1.0,
		SomeFloat64: -2.0,
		SomeInt:     -1,
		SomeInt8:    -2,
		SomeInt16:   -3,
		SomeInt32:   -4,
		SomeInt64:   -5,
		SomeUint8:   6,
		SomeUint16:  7,
		SomeUint32:  8,
		SomeUint64:  9,
	}
	result, err := twoWaysGeneric[EmptyStruct, PrimitiveTypesStructWithDefault](t, sample)
	assert.NilError(t, err)
	assert.Equal(t, *result, expected, "Deserialization should have inserted default values")
}

// Test that default values are parsed.
func TestStructDefaultValues(t *testing.T) {
	type PairWithDefaults[T any, U any] struct {
		Left  T `default:"{}"`
		Right U `default:"{}"`
	}
	sample := EmptyStruct{}
	expected := PairWithDefaults[EmptyStruct, EmptyStruct]{
		Left:  EmptyStruct{},
		Right: EmptyStruct{},
	}
	result, err := twoWaysGeneric[EmptyStruct, PairWithDefaults[EmptyStruct, EmptyStruct]](t, sample)
	assert.NilError(t, err)
	assert.Equal(t, *result, expected, "Deserialization should have inserted default values")
}

func TestStructDefaultValuesInvalidSyntax(t *testing.T) {
	type PairWithDefaults[T any, U any] struct {
		Left  T `default:"{}"`
		Right U `default:"{}"`
	}
	_, err := deserialize.MakeMapDeserializer[PairWithDefaults[PairWithDefaults[EmptyStruct, int], int]](deserialize.JSONOptions(""))

	assert.ErrorContains(t, err, "could not generate a deserializer")
	assert.ErrorContains(t, err, "cannot parse default value")
	assert.ErrorContains(t, err, "strconv.Atoi: parsing \"{}\": invalid syntax")
}

// Check that when we have a default struct of {}, we're still going to
// use the default values within that struct.
func TestStructDefaultValuesNestedStruct(t *testing.T) {
	type PairWithDefaults[T any, U any] struct {
		Left  T `default:"{}"`
		Right U `default:"{}"`
	}
	type SimpleStructWithDefaults struct {
		SomeString string `default:"some string"`
	}
	sample := EmptyStruct{}
	expected := PairWithDefaults[PairWithDefaults[SimpleStructWithDefaults, SimpleStructWithDefaults], SimpleStructWithDefaults]{
		Left: PairWithDefaults[SimpleStructWithDefaults, SimpleStructWithDefaults]{
			Left: SimpleStructWithDefaults{
				SomeString: "some string",
			},
			Right: SimpleStructWithDefaults{
				SomeString: "some string",
			},
		},
		Right: SimpleStructWithDefaults{
			SomeString: "some string",
		},
	}
	result, err := twoWaysGeneric[EmptyStruct, PairWithDefaults[PairWithDefaults[SimpleStructWithDefaults, SimpleStructWithDefaults], SimpleStructWithDefaults]](t, sample)
	assert.NilError(t, err)
	assert.Equal(t, *result, expected, "We should have inserted default values recursively")
}

// Test that default values are parsed.
func TestSliceDefaultValues(t *testing.T) {
	type ArrayWithDefault struct {
		Data  []EmptyStruct  `default:"[]"`
		Data2 [0]EmptyStruct `default:"[]"`
	}
	result, err := twoWaysGeneric[EmptyStruct, ArrayWithDefault](t, EmptyStruct{})
	assert.NilError(t, err)
	assert.Equal(t, len(result.Data), 0, "Deserialization should have inserted default values")
}

// ---- Test that a bad default value will cause an error while generating the deserializer.

func TestBadDefaultValues(t *testing.T) {
	type PrimitiveTypesStructWithDefault struct {
		SomeBool    bool    `default:"blue"`
		SomeFloat32 float32 `default:"blue"`
		SomeFloat64 float64 `default:"blue"`
		SomeInt     int     `default:"blue"`
		SomeInt8    int8    `default:"blue"`
		SomeInt16   int16   `default:"blue"`
		SomeInt32   int32   `default:"blue"`
		SomeInt64   int64   `default:"blue"`
		SomeUint8   uint8   `default:"blue"`
		SomeUint16  uint16  `default:"blue"`
		SomeUint32  uint32  `default:"blue"`
		SomeUint64  uint64  `default:"blue"`
	}
	_, err := deserialize.MakeMapDeserializer[PrimitiveTypesStructWithDefault](deserialize.JSONOptions("")) //nolint:exhaustruct
	assert.ErrorContains(t, err, "cannot parse default value at PrimitiveTypesStruct", "Deserialization should have parsed strings")
}

// ------

// ---- Test that a good default value will succeed.

type SimpleStructWithOrMethodSuccess struct {
	SomeString string            `orMethod:"MakeString"`
	SomeSlice  []string          `orMethod:"MakeStringSlice"`
	SomeMap    map[string]string `orMethod:"MakeMap"`
}

func (SimpleStructWithOrMethodSuccess) MakeString() (string, error) {
	return "This is a string", nil
}

func (SimpleStructWithOrMethodSuccess) MakeStringSlice() ([]string, error) {
	return []string{"This is a slice"}, nil
}

func (SimpleStructWithOrMethodSuccess) MakeMap() (map[string]string, error) {
	return map[string]string{"zero": "This is a map"}, nil
}

// Test that an `orMethod` will be called if no value is provided (success case).
func TestOrMethodSuccess(t *testing.T) {
	expected := SimpleStructWithOrMethodSuccess{
		SomeString: "This is a string",
		SomeSlice:  []string{"This is a slice"},
		SomeMap:    map[string]string{"zero": "This is a map"},
	}
	result, err := twoWaysGeneric[EmptyStruct, SimpleStructWithOrMethodSuccess](t, EmptyStruct{})
	assert.NilError(t, err)
	assert.DeepEqual(t, *result, expected)
}

// ----

// ----  Test that an `orMethod` will be called if no value is provided (error case). ---

type SimpleStructWithFlatOrMethodError struct {
	SomeString string `orMethod:"MakeString"`
}

func (SimpleStructWithFlatOrMethodError) MakeString() (string, error) {
	return "Test value constructed with a method", errors.New("This is an error from SimpleStructWithFlatOrMethodError")
}

type SimpleStructWithPtrOrMethodError struct {
	SomeString *string `orMethod:"MakeString"`
}

func (SimpleStructWithPtrOrMethodError) MakeString() (*string, error) {
	return nil, errors.New("This is an error from SimpleStructWithPtrOrMethodError")
}

type SimpleStructWithSliceOrMethodError struct {
	SomeString []string `orMethod:"MakeStringSlice"`
}

func (SimpleStructWithSliceOrMethodError) MakeStringSlice() ([]string, error) {
	return []string{"Test value constructed with a method"}, errors.New("This is an error from SimpleStructWithSliceOrMethodError")
}

type SimpleStructWithMapOrMethodError struct {
	SomeString map[string]string `orMethod:"MakeStringMap"`
}

func (SimpleStructWithMapOrMethodError) MakeStringMap() (map[string]string, error) {
	return map[string]string{"zero": "Test value constructed with a method"}, errors.New("This is an error from SimpleStructWithMapOrMethodError")
}

type SimpleStructWithStructOrMethodError struct {
	SomeString SimpleStruct `orMethod:"MakeSimpleStruct"`
}

func (SimpleStructWithStructOrMethodError) MakeSimpleStruct() (SimpleStruct, error) {
	return SimpleStruct{}, errors.New("This is an error from SimpleStructWithStructOrMethodError")
}

func TestOrMethodError(t *testing.T) {
	_, err := twoWaysGeneric[EmptyStruct, SimpleStructWithFlatOrMethodError](t, EmptyStruct{})
	assert.Equal(t, err.Error(), "error in optional value at SimpleStructWithFlatOrMethodError.SomeString\n\t * This is an error from SimpleStructWithFlatOrMethodError", "The method should have been called to inject a value")

	_, err = twoWaysGeneric[EmptyStruct, SimpleStructWithPtrOrMethodError](t, EmptyStruct{})
	assert.Equal(t, err.Error(), "error in optional value at SimpleStructWithPtrOrMethodError.SomeString\n\t * This is an error from SimpleStructWithPtrOrMethodError", "The method should have been called to inject a value")

	_, err = twoWaysGeneric[EmptyStruct, SimpleStructWithSliceOrMethodError](t, EmptyStruct{})
	assert.Equal(t, err.Error(), "error in optional value at SimpleStructWithSliceOrMethodError.SomeString\n\t * This is an error from SimpleStructWithSliceOrMethodError", "The method should have been called to inject a value")

	_, err = twoWaysGeneric[EmptyStruct, SimpleStructWithStructOrMethodError](t, EmptyStruct{})
	assert.Equal(t, err.Error(), "error in optional value at SimpleStructWithStructOrMethodError.SomeString\n\t * This is an error from SimpleStructWithStructOrMethodError", "The method should have been called to inject a value")

	_, err = twoWaysGeneric[EmptyStruct, SimpleStructWithMapOrMethodError](t, EmptyStruct{})
	assert.Equal(t, err.Error(), "error in optional value at SimpleStructWithMapOrMethodError.SomeString\n\t * This is an error from SimpleStructWithMapOrMethodError", "The method should have been called to inject a value")
}

// ----------

// Test error cases for `onMethod` setup.

type SimpleStructWithOrMethodBadName struct {
	SomeString string `orMethod:"IDoNotExist"`
}
type SimpleStructWithOrMethodBadArgs struct {
	SomeString string `orMethod:"BadArgs"`
}

func (SimpleStructWithOrMethodBadArgs) BadArgs(string) (string, error) {
	// This method should not take any arguments.
	return "", nil
}

type SimpleStructWithOrMethodBadOut1 struct {
	SomeInt int `orMethod:"BadOut1"`
}

func (SimpleStructWithOrMethodBadOut1) BadOut1() (string, error) {
	return "0", nil
}

type SimpleStructWithOrMethodBadOut2 struct {
	SomeString string `orMethod:"BadOut2"`
}

func (SimpleStructWithOrMethodBadOut2) BadOut2() (string, string) {
	return "", ""
}

type SimpleStructWithOrMethodBadOut3 struct {
	SomeString string `orMethod:"BadOut3"`
}

func (SimpleStructWithOrMethodBadOut3) BadOut3() (string, error, error) {
	return "", nil, nil
}

type SimpleStructWithOrMethodMissingMethod struct {
	SomeString string `orMethod:"IDoNotExist"`
}

func TestOrMethodBadSetup(t *testing.T) {
	_, err := deserialize.MakeMapDeserializer[SimpleStructWithOrMethodBadName](deserialize.JSONOptions("")) //nolint:exhaustruct
	assert.Equal(t, err.Error(), "could not generate a deserializer for SimpleStructWithOrMethodBadName.SomeString with type string:\n\t * at SimpleStructWithOrMethodBadName.SomeString, failed to setup `orMethod`\n\t * method IDoNotExist provided with `orMethod` doesn't seem to exist - note that the method must be public", "We should fail early if the orMethod doesn't exist")

	_, err = deserialize.MakeMapDeserializer[SimpleStructWithOrMethodBadArgs](deserialize.JSONOptions(""))
	assert.Equal(t, err.Error(), "could not generate a deserializer for SimpleStructWithOrMethodBadArgs.SomeString with type string:\n\t * at SimpleStructWithOrMethodBadArgs.SomeString, failed to setup `orMethod`\n\t * the method provided with `orMethod` MUST take no argument but takes 1 arguments", "We should fail early if orMethod args are incorrect")

	_, err = deserialize.MakeMapDeserializer[SimpleStructWithOrMethodBadOut1](deserialize.JSONOptions(""))
	assert.Equal(t, err.Error(), "could not generate a deserializer for SimpleStructWithOrMethodBadOut1.SomeInt with type int:\n\t * at SimpleStructWithOrMethodBadOut1.SomeInt, failed to setup `orMethod`\n\t * the method provided with `orMethod` MUST return (int, error) but it returns (string, _) which is not convertible to `int`", "We should fail early if first result is incorrect")

	_, err = deserialize.MakeMapDeserializer[SimpleStructWithOrMethodBadOut2](deserialize.JSONOptions(""))
	assert.Equal(t, err.Error(), "could not generate a deserializer for SimpleStructWithOrMethodBadOut2.SomeString with type string:\n\t * at SimpleStructWithOrMethodBadOut2.SomeString, failed to setup `orMethod`\n\t * the method provided with `orMethod` MUST return (string, error) but it returns (_, string) which is not convertible to `error`", "We should fail early if second result is incorrect")

	_, err = deserialize.MakeMapDeserializer[SimpleStructWithOrMethodBadOut3](deserialize.JSONOptions(""))
	assert.Equal(t, err.Error(), "could not generate a deserializer for SimpleStructWithOrMethodBadOut3.SomeString with type string:\n\t * at SimpleStructWithOrMethodBadOut3.SomeString, failed to setup `orMethod`\n\t * the method provided with `orMethod` MUST return (string, error) but it returns 3 value(s)", "We should fail early if second result is incorrect")

	_, err = deserialize.MakeMapDeserializer[SimpleStructWithOrMethodMissingMethod](deserialize.JSONOptions(""))
	assert.Equal(t, err.Error(), "could not generate a deserializer for SimpleStructWithOrMethodMissingMethod.SomeString with type string:\n\t * at SimpleStructWithOrMethodMissingMethod.SomeString, failed to setup `orMethod`\n\t * method IDoNotExist provided with `orMethod` doesn't seem to exist - note that the method must be public", "We should fail early if second result is incorrect")
}

// -------

type NestedStructWithOrMethod struct {
	SomeStruct SimpleStruct `orMethod:"MakeSimpleStruct"`
}

func (NestedStructWithOrMethod) MakeSimpleStruct() (SimpleStruct, error) {
	return SimpleStruct{
		SomeString: "I have been made",
	}, nil
}

func TestOrMethodStruct(t *testing.T) {
	expected := NestedStructWithOrMethod{
		SomeStruct: SimpleStruct{
			SomeString: "I have been made",
		},
	}
	result, err := twoWaysGeneric[EmptyStruct, NestedStructWithOrMethod](t, EmptyStruct{})
	assert.NilError(t, err)
	assert.Equal(t, *result, expected, "The method should have been called to inject a value")
}

type NestedSliceWithOrMethod struct {
	SomeSlice []SimpleStruct `orMethod:"MakeSimpleSlice"`
}

func (NestedSliceWithOrMethod) MakeSimpleSlice() ([]SimpleStruct, error) {
	result := []SimpleStruct{{
		SomeString: "I have been made 1",
	}, {
		SomeString: "I have been made 2",
	}}
	return result, nil
}

func TestOrMethodSlice(t *testing.T) {
	expected := NestedSliceWithOrMethod{}
	expected.SomeSlice, _ = expected.MakeSimpleSlice()
	result, err := twoWaysGeneric[EmptyStruct, NestedSliceWithOrMethod](t, EmptyStruct{})
	assert.NilError(t, err)
	assert.DeepEqual(t, result.SomeSlice, expected.SomeSlice)
}

type StructWithPointer struct {
	SomePointer *SimpleStruct
}

func TestDeserializePointer(t *testing.T) {
	simpleStruct := SimpleStruct{
		SomeString: "some string",
	}
	sample := StructWithPointer{
		SomePointer: &simpleStruct,
	}
	result, err := twoWays(t, sample)
	assert.NilError(t, err)
	assert.Equal(t, result.SomePointer.SomeString, sample.SomePointer.SomeString, "We should be able to deserialize with pointers")
}

func TestPointerDefaultValue(t *testing.T) {
	type StructWithPointerDefault struct {
		SomePointer *SimpleStruct `default:"nil"`
	}
	result, err := twoWaysGeneric[EmptyStruct, StructWithPointerDefault](t, EmptyStruct{})
	assert.NilError(t, err)
	var expected *SimpleStruct /* = nil */
	assert.Equal(t, result.SomePointer, expected, "We should be able to deserialize with pointers using a default nil")
}

type StructWithPointerOrMethod struct {
	SomePointer *SimpleStruct `orMethod:"MakeSimpleStructPtr"`
}

func (StructWithPointerOrMethod) MakeSimpleStructPtr() (*SimpleStruct, error) {
	result := SimpleStruct{
		SomeString: "I have been made (behind a pointer)",
	}
	return &result, nil
}

func TestPointerOrMethod(t *testing.T) {
	sample := new(StructWithPointerOrMethod)
	sample.SomePointer, _ = sample.MakeSimpleStructPtr()
	result, err := twoWaysGeneric[EmptyStruct, StructWithPointerOrMethod](t, EmptyStruct{})
	assert.NilError(t, err)
	assert.Equal(t, result.SomePointer.SomeString, sample.SomePointer.SomeString, "We should be able to deserialize with pointers using a orMethod")
}

type StructWithInitializer struct {
	SomeBool    bool
	SomeString  string
	SomeFloat32 float32
	SomeFloat64 float64
	SomeInt     int
	SomeInt8    int8
	SomeInt16   int16
	SomeInt32   int32
	SomeInt64   int64
	SomeUint8   uint8
	SomeUint16  uint16
	SomeUint32  uint32
	SomeUint64  uint64
}

func (s *StructWithInitializer) Initialize() error {
	s.SomeBool = true
	s.SomeString = "text"
	s.SomeFloat32 = -1.0
	s.SomeFloat64 = -2.0
	s.SomeInt = -1
	s.SomeInt8 = -2
	s.SomeInt16 = -3
	s.SomeInt32 = -4
	s.SomeInt64 = -5
	s.SomeUint8 = 6
	s.SomeUint16 = 7
	s.SomeUint32 = 8
	s.SomeUint64 = 9
	return nil
}

var _ validation.Initializer = &StructWithInitializer{} // Type assertion.

// Test that we're correctly running pre-initialization at toplevel.
func TestInitializer(t *testing.T) {
	sample := EmptyStruct{}
	expected := StructWithInitializer{}
	_ = expected.Initialize()
	result, err := twoWaysGeneric[EmptyStruct, StructWithInitializer](t, sample)
	assert.NilError(t, err)
	assert.Equal(t, *result, expected, "Initializer should have initialized our structure")
}

// Test that we're correctly running pre-initialization within another struct.
func TestInitializerNested(t *testing.T) {
	sample := Pair[EmptyStruct, EmptyStruct]{
		Left:  EmptyStruct{},
		Right: EmptyStruct{},
	}
	expected := Pair[EmptyStruct, StructWithInitializer]{
		Left:  EmptyStruct{},
		Right: StructWithInitializer{},
	}
	_ = expected.Right.Initialize()
	result, err := twoWaysGeneric[Pair[EmptyStruct, EmptyStruct], Pair[EmptyStruct, StructWithInitializer]](t, sample)
	assert.NilError(t, err)
	assert.Equal(t, *result, expected, "Initializer should have initialized our structure")
}

// ----- Test that we're correctly running pre-initialization.

type StructInitializerFaulty struct {
	SomeBool    bool
	SomeString  string
	SomeFloat32 float32
	SomeFloat64 float64
	SomeInt     int
	SomeInt8    int8
	SomeInt16   int16
	SomeInt32   int32
	SomeInt64   int64
	SomeUint8   uint8
	SomeUint16  uint16
	SomeUint32  uint32
	SomeUint64  uint64
}

func (s *StructInitializerFaulty) Initialize() error {
	s.SomeBool = true
	s.SomeString = "text"
	s.SomeFloat32 = -1.0
	s.SomeFloat64 = -2.0
	s.SomeInt = -1
	s.SomeInt8 = -2
	s.SomeInt16 = -3
	s.SomeInt32 = -4
	s.SomeInt64 = -5
	s.SomeUint8 = 6
	s.SomeUint16 = 7
	s.SomeUint32 = 8
	s.SomeUint64 = 9
	return errors.New("Test error")
}

var _ validation.Initializer = &StructInitializerFaulty{} // Type assertion.

func TestInitializerFaulty(t *testing.T) {
	sample := EmptyStruct{}
	_, err := twoWaysGeneric[EmptyStruct, StructInitializerFaulty](t, sample)
	assert.Equal(t, err.Error(), "at StructInitializerFaulty, encountered an error while initializing optional fields:\n\t * Test error", "Initializer should have initialized our structure")

	asCustom := deserialize.CustomDeserializerError{}
	ok := errors.As(err, &asCustom)
	assert.Equal(t, ok, true, "the error should be a CustomDeserializerError")

	type ContainerStructFaulty struct {
		Data StructInitializerFaulty
	}
	_, err = twoWaysGeneric[EmptyStruct, ContainerStructFaulty](t, sample)
	assert.ErrorContains(t, err, "at ContainerStructFaulty.Data, encountered an error while initializing optional fields:\n\t * Test error", "Initializer should have initialized our structure")

	ok = errors.As(err, &asCustom)
	assert.Equal(t, ok, true, "the error should be a CustomDeserializerError")
}

// -----

type StructUnmarshal struct {
	hidden string
}

func (su *StructUnmarshal) UnmarshalJSON(source []byte) error {
	if len(source) == 0 {
		return errors.New("Test error: this slice is too short")
	}
	str := string(source)
	su.hidden = str
	return nil
}

func (su StructUnmarshal) MarshalJSON() ([]byte, error) {
	return json.Marshal(su.hidden) //nolint:wrapcheck
}

var _ json.Unmarshaler = &StructUnmarshal{} // Type assertion.
var _ json.Marshaler = StructUnmarshal{}    // Type assertion.

func TestUnmarshal(t *testing.T) {
	sample := Pair[int, StructUnmarshal]{
		Left: 123,
		Right: StructUnmarshal{
			hidden: "Hidden field",
		},
	}
	result, err := twoWays(t, sample)
	assert.NilError(t, err)
	assert.Equal(t, *result, sample, "We should have deserialized correctly through UnmarshalJSON")
}

// A data structure that implements Validate but forgets the pointer.
type BadValidateStruct struct{}

func (BadValidateStruct) Validate() error { // Should be `func (BadValidateStruct*) Validate() error`
	return nil
}

var _ validation.Validator = BadValidateStruct{}

// A data structure that implements Initialize but forgets the pointer.
type BadInitializeStruct struct{}

func (BadInitializeStruct) Initialize() error { // Should be `func (BadInitializeStruct*) Initialize() error`
	return nil
}

var _ validation.Initializer = BadInitializeStruct{}

// Test that we can detect bad implementations of `Validator` or `Initializer` that do not
// work on pointers.
func TestBadValidate(t *testing.T) {
	_, err := deserialize.MakeMapDeserializer[BadValidateStruct](deserialize.JSONOptions("")) //nolint:exhaustruct
	assert.Equal(t, err.Error(), "type deserialize_test.BadValidateStruct implements validation.Validator - it should be implemented by pointer type *deserialize_test.BadValidateStruct instead", "We should have detected that this struct does not implement Validator correctly")

	_, err = deserialize.MakeMapDeserializer[BadInitializeStruct](deserialize.JSONOptions("")) //nolint:exhaustruct
	assert.Equal(t, err.Error(), "type deserialize_test.BadInitializeStruct implements validation.Initializer - it should be implemented by pointer type *deserialize_test.BadInitializeStruct instead", "We should have detected that this struct does not implement Initializer correctly")
}

type StructMap[T any] struct {
	RegularMap  map[string]T
	DefaultMap  map[string]T `default:"{}"`
	OrMethodMap map[string]T `orMethod:"SetupMap"`
}

func (StructMap[T]) SetupMap() (map[string]T, error) {
	result := make(map[string]T)
	return result, nil
}

func TestMap(t *testing.T) {
	type SimpleStructWithDefault struct {
		SomeString string `default:"default value"`
	}

	_, err := twoWaysGeneric[EmptyStruct, StructMap[SimpleStructWithDefault]](t, EmptyStruct{})
	assert.ErrorContains(t, err, "missing object value at StructMap", "We should have failed because the map is missing a default or a value")

	sampleMap := map[string]int{"zero": 0, "one": 1, "two": 2}
	expected := StructMap[int]{
		RegularMap:  sampleMap,
		DefaultMap:  map[string]int{},
		OrMethodMap: map[string]int{},
	}
	extracted, err := twoWaysGeneric[StructMap[int], StructMap[int]](t, StructMap[int]{
		RegularMap: sampleMap,
	})
	if err != nil {
		t.Error(err)

		return
	}
	assert.DeepEqual(t, *extracted, expected)
}

func TestBadMapMap(t *testing.T) {
	type BadMap struct {
		Field map[int]string
	}

	_, err := deserialize.MakeMapDeserializer[BadMap](deserialize.JSONOptions(""))
	assert.ErrorContains(t, err, "invalid map type at BadMap.Field, only map[string]T can be converted into a deserializer", "We should have detected that we cannot convert maps with such keys")
}
func TestInvalidDefaultValues(t *testing.T) {
	type RandomStruct struct {
	}
	type InvalidStruct struct {
		Field RandomStruct `default:"abc"`
	}
	type InvalidPointer struct {
		Field *RandomStruct `default:"abc"`
	}
	type InvalidSlice struct {
		Field []RandomStruct `default:"abc"`
	}
	type InvalidArray struct {
		Field [4]RandomStruct `default:"abc"`
	}
	_, err := deserialize.MakeMapDeserializer[InvalidStruct](deserialize.JSONOptions(""))
	assert.ErrorContains(t, err, "invalid `default` value", "We should have detected that we cannot convert maps with such keys")

	_, err = deserialize.MakeMapDeserializer[InvalidPointer](deserialize.JSONOptions(""))
	assert.ErrorContains(t, err, "invalid `default` value", "We should have detected that we cannot convert maps with such keys")

	_, err = deserialize.MakeMapDeserializer[InvalidSlice](deserialize.JSONOptions(""))
	assert.ErrorContains(t, err, "invalid `default` value", "We should have detected that we cannot convert maps with such keys")

	_, err = deserialize.MakeMapDeserializer[InvalidArray](deserialize.JSONOptions(""))
	assert.ErrorContains(t, err, "invalid `default` value", "We should have detected that we cannot convert maps with such keys")

}

func TestInvalidStruct(t *testing.T) {
	_, err := deserialize.MakeKVListDeserializer[int](deserialize.JSONOptions(""))
	assert.ErrorContains(t, err, "invalid call to StructDeserializer: int is not a struct")

	_, err = deserialize.MakeMapDeserializer[int](deserialize.JSONOptions(""))
	assert.ErrorContains(t, err, "invalid call to StructDeserializer: int is not a struct")
}

func TestInvalidTags(t *testing.T) {
	type BadTag struct {
		Field string `json:"abc" json:"abc"`
	}
	_, err := deserialize.MakeKVListDeserializer[BadTag](deserialize.JSONOptions(""))
	assert.ErrorContains(t, err, "invalid tag, name json should only be defined once")
}

// ------ Test that when a struct implements both `Unmarshaler` and `Initializer`, we
// default to `Unmarshaler`.

type StructSupportBothUnmarshalerAndInitializer struct {
	Field string
}

func (s *StructSupportBothUnmarshalerAndInitializer) Initialize() error {
	panic("this method shouldn't have been called")
}
func (s *StructSupportBothUnmarshalerAndInitializer) UnmarshalJSON(buf []byte) error {
	container := map[string]string{}
	err := json.Unmarshal(buf, &container)
	if err != nil {
		panic(err)
	}

	s.Field = "test has succeeded with " + container["Field"]
	return nil
}

func TestSupportBothUnmarshalerAndInitializer(t *testing.T) {
	sample := StructSupportBothUnmarshalerAndInitializer{
		Field: "some content",
	}
	result, err := twoWays[StructSupportBothUnmarshalerAndInitializer](t, sample)
	assert.NilError(t, err)
	assert.DeepEqual(t, result, &StructSupportBothUnmarshalerAndInitializer{Field: "test has succeeded with some content"})
}

// -------

// ------ Test that when a struct implements both `Unmarshaler` and `DictUnmarshaler`, we
// default to `DictUnmarshaler`.

type StructSupportBothUnmarshalerAndDictUnmarshaler struct {
	Field string
}

func (s *StructSupportBothUnmarshalerAndDictUnmarshaler) UnmarshalJSON([]byte) error {
	panic("this method shouldn't have been called")
}

func (s *StructSupportBothUnmarshalerAndDictUnmarshaler) UnmarshalDict(in shared.Dict) error {
	value, ok := in.Lookup("Field")
	if !ok {
		panic("missing key")
	}
	str, ok := value.Interface().(string)
	if !ok {
		panic("invalid value")
	}

	s.Field = "test has succeeded with " + str
	return nil
}

var _ shared.UnmarshalDict = new(StructSupportBothUnmarshalerAndDictUnmarshaler)

func TestSupportBothUnmarshalerAndDictInitializer(t *testing.T) {
	sample := StructSupportBothUnmarshalerAndDictUnmarshaler{
		Field: "some content",
	}
	result, err := twoWays[StructSupportBothUnmarshalerAndDictUnmarshaler](t, sample)
	assert.NilError(t, err)
	assert.DeepEqual(t, result, &StructSupportBothUnmarshalerAndDictUnmarshaler{Field: "test has succeeded with some content"})
}

// -----

// ----- Test that we can deserialize a struct with a field that should not be deserializable if we have some kind of pre-initializer.

type StructThatCannotBeDeserialized struct {
	private bool
}

type StructThatCannotBeDeserialized2 struct {
	private bool
}

func (*StructThatCannotBeDeserialized2) Initialize() error {
	return nil
}

type StructWithTime struct {
	Field  StructThatCannotBeDeserialized `initialized:"true"`
	Field2 StructThatCannotBeDeserialized2
	Field3 time.Time
}

func TestDeserializingWithPreinitializer(t *testing.T) {
	date := time.Date(2000, 01, 01, 01, 01, 01, 01, time.UTC)
	sample := StructWithTime{
		Field:  StructThatCannotBeDeserialized{private: false},
		Field2: StructThatCannotBeDeserialized2{private: false},
		Field3: date,
	}
	result, err := twoWays[StructWithTime](t, sample)
	assert.NilError(t, err)
	assert.DeepEqual(t, result, &sample, cmpopts.IgnoreUnexported(StructThatCannotBeDeserialized{}, StructThatCannotBeDeserialized2{}))
}

// ------

// ------ Test that we can deserialize uuid through json.

type TextUnmarshalerUUID uuid.UUID

func (t *TextUnmarshalerUUID) UnmarshalText(source []byte) error {
	result, err := uuid.Parse(string(source))
	if err != nil {
		return err //nolint:wrapcheck
	}
	*t = TextUnmarshalerUUID(result)
	return nil
}

var _ encoding.TextUnmarshaler = &TextUnmarshalerUUID{}

type StructWithUUID struct {
	Field TextUnmarshalerUUID
}

func TestDeserializeUUIDJSON(t *testing.T) {
	sample := StructWithUUID{
		Field: TextUnmarshalerUUID(uuid.New()),
	}
	result, err := twoWays[StructWithUUID](t, sample)
	assert.NilError(t, err)
	assert.DeepEqual(t, result, &sample)
}

// ------

// ------ Test that we can deserialize uuid through kvlist.

func TestDeserializeUUIDKV(t *testing.T) {
	sample := StructWithUUID{
		Field: TextUnmarshalerUUID(uuid.New()),
	}
	deserializer, err := deserialize.MakeKVListDeserializer[StructWithUUID](deserialize.QueryOptions(""))
	assert.NilError(t, err)

	kvList := map[string][]string{}
	kvList["Field"] = []string{uuid.UUID(sample.Field).String()}
	deserialized, err := deserializer.DeserializeKVList(kvList)
	assert.NilError(t, err)

	assert.Equal(t, deserialized.Field, sample.Field)
}

// ------

// ------ Test that we can pass nil where a pointer is expected.

func TestNilPtr(t *testing.T) {
	type Struct struct {
		Field *string
	}
	sample := Struct{
		Field: nil,
	}
	found, err := twoWays(t, sample)
	assert.NilError(t, err)
	assert.DeepEqual(t, *found, sample)
}

// ------

// ------ Test that we can deserialize a value with a private type

type Private uint

type StructWithPrivate struct {
	Field Private
}

func TestMapDeserializeWithPrivate(t *testing.T) {
	sample := StructWithPrivate{
		Field: 265,
	}

	deserialized, err := twoWays(t, sample)
	assert.NilError(t, err)
	assert.Equal(t, *deserialized, sample)
}

func TestKVDeserializeWithPrivate(t *testing.T) {
	sample := StructWithPrivate{
		Field: 265,
	}
	deserializer, err := deserialize.MakeKVListDeserializer[StructWithPrivate](deserialize.QueryOptions(""))

	assert.NilError(t, err)
	kvList := map[string][]string{}
	kvList["Field"] = []string{fmt.Sprint(sample.Field)}

	deserialized, err := deserializer.DeserializeKVList(kvList)
	assert.NilError(t, err)
	assert.Equal(t, *deserialized, sample)
}

// ------ Test that we can deserialize things more complicated than just `[]string` with KVList

type StructWithPrimitiveSlices struct {
	SomeStrings []string
	SomeInts    []int
	SomeInt8    []int8
	SomeInt16   []int16
	SomeInt32   []int32
	SomeInt64   []int64
	SomeUints   []uint
	SomeUint8   []uint8
	SomeUint16  []uint16
	SomeUint32  []uint32
	SomeUint64  []uint64
	SomeBools   []bool
	SomeFloat32 []float32
	SomeFloat64 []float64
}

func TestKVDeserializePrimitiveSlices(t *testing.T) {
	deserializer, err := deserialize.MakeKVListDeserializer[StructWithPrimitiveSlices](deserialize.QueryOptions(""))
	assert.NilError(t, err)

	sample := StructWithPrimitiveSlices{
		SomeStrings: []string{"abc", "def"},
		SomeInts:    []int{15, 0, -15},
		SomeInt8:    []int8{0, -2, 4, 8},
		SomeInt16:   []int16{16, -32, 64},
		SomeInt32:   []int32{128, -256, 512},
		SomeInt64:   []int64{1024, -2048, 4096},
		SomeUints:   []uint{0, 2, 4, 8},
		SomeUint8:   []uint8{16, 32, 64, 128},
		SomeUint16:  []uint16{256, 512, 1024, 2048},
		SomeUint32:  []uint32{4096, 8192, 16364},
		SomeUint64:  []uint64{32768, 65536},
		SomeBools:   []bool{true, true, false, true},
		SomeFloat32: []float32{3.1415, 1.2},
		SomeFloat64: []float64{42.0},
	}

	kvlist := make(map[string][]string, 0)

	kvlist["SomeStrings"] = []string{"abc", "def"}
	kvlist["SomeInts"] = []string{"15", "0", "-15"}
	kvlist["SomeInt8"] = []string{"0", "-2", "4", "8"}
	kvlist["SomeInt16"] = []string{"16", "-32", "64"}
	kvlist["SomeInt32"] = []string{"128", "-256", "512"}
	kvlist["SomeInt64"] = []string{"1024", "-2048", "4096"}
	kvlist["SomeUints"] = []string{"0", "2", "4", "8"}
	kvlist["SomeUint8"] = []string{"16", "32", "64", "128"}
	kvlist["SomeUint16"] = []string{"256", "512", "1024", "2048"}
	kvlist["SomeUint32"] = []string{"4096", "8192", "16364"}
	kvlist["SomeUint64"] = []string{"32768", "65536"}
	kvlist["SomeBools"] = []string{"true", "true", "false", "true"}
	kvlist["SomeFloat32"] = []string{"3.1415", "1.2"}
	kvlist["SomeFloat64"] = []string{"42.0"}

	deserialized, err := deserializer.DeserializeKVList(kvlist)
	assert.NilError(t, err)
	assert.DeepEqual(t, *deserialized, sample)

}

func TestDeserializeUUIDKVList(t *testing.T) {
	deserializer, err := deserialize.MakeKVListDeserializer[StructWithUUID](deserialize.QueryOptions(""))
	assert.NilError(t, err)

	// This is deserializable because the field supports `TextUnmarshal`
	sample := StructWithUUID{
		Field: TextUnmarshalerUUID(uuid.New()),
	}

	marshaledField, err := uuid.UUID(sample.Field).MarshalText()
	assert.NilError(t, err)
	kvlist := make(map[string][]string, 0)
	kvlist["Field"] = []string{string(marshaledField)}

	deserialized, err := deserializer.DeserializeKVList(kvlist)
	assert.NilError(t, err)
	assert.DeepEqual(t, *deserialized, sample)
}

// ------ Test that KVList detects structures that it cannot deserialize

// A struct that just can't be deserialized.
type StructWithChan struct {
	Chan chan int
}

func TestKVCannotDeserializeChan(t *testing.T) {
	_, err := deserialize.MakeKVListDeserializer[StructWithChan](deserialize.QueryOptions(""))
	if err == nil {
		t.Fatal("this should have failed")
	}
	assert.ErrorContains(t, err, "chan int")
}

// ------ Test that KVList calls validation

type CustomStructWithValidation struct {
	Field int
}

func (c *CustomStructWithValidation) Validate() error {
	if c.Field < 0 {
		return errors.New("custom validation error")
	}
	return nil
}

func (c *CustomStructWithValidation) UnmarshalText(source []byte) error {
	result, err := strconv.Atoi(string(source))
	if err != nil {
		return err //nolint:wrapcheck
	}
	c.Field = result
	return nil
}

func TestKVCallsInnerValidation(t *testing.T) {
	type Struct struct {
		Inner CustomStructWithValidation
	}
	deserializer, err := deserialize.MakeKVListDeserializer[Struct](deserialize.QueryOptions(""))
	assert.NilError(t, err)

	goodSample := Struct{
		Inner: CustomStructWithValidation{
			Field: 123,
		},
	}

	kvlist := make(map[string][]string, 0)
	kvlist["Inner"] = []string{strconv.Itoa(goodSample.Inner.Field)}

	deserialized, err := deserializer.DeserializeKVList(kvlist)
	assert.NilError(t, err)
	assert.DeepEqual(t, *deserialized, goodSample)

	badSample := Struct{
		Inner: CustomStructWithValidation{
			Field: -123,
		},
	}

	kvlist["Inner"] = []string{strconv.Itoa(badSample.Inner.Field)}

	_, err = deserializer.DeserializeKVList(kvlist)
	assert.ErrorContains(t, err, "custom validation error")
}

// ------ Test that flattened structs are deserialized properly.
func TestMapDeserializerFlattened(t *testing.T) {
	type Inner struct {
		Left  string
		Right string
	}
	type Outer struct {
		Flattened Inner `flatten:""`
		Inner
		Regular Inner
	}

	deserializer, err := deserialize.MakeMapDeserializer[Outer](deserialize.JSONOptions(""))
	assert.NilError(t, err)

	data := `
	{
		"Left": "flattened_left",
		"Right": "flattened_right",
		"Regular": {
			"Left": "regular_left",
			"Right": "regular_right"
		}
	}`
	expected := Outer{
		Flattened: Inner{
			Left:  "flattened_left",
			Right: "flattened_right",
		},
		Inner: Inner{
			Left:  "flattened_left",
			Right: "flattened_right",
		},
		Regular: Inner{
			Left:  "regular_left",
			Right: "regular_right",
		},
	}
	found, err := deserializer.DeserializeBytes([]byte(data))
	assert.NilError(t, err)

	assert.DeepEqual(t, *found, expected)
}

func TestKVDeserializerFlattened(t *testing.T) {
	type Inner struct {
		Left  string
		Right string
	}
	type Outer struct {
		Flattened Inner `flatten:""`
		Inner
	}

	deserializer, err := deserialize.MakeKVListDeserializer[Outer](deserialize.QueryOptions(""))
	assert.NilError(t, err)

	data := make(map[string][]string)
	data["Left"] = []string{"flattened_left"}
	data["Right"] = []string{"flattened_right"}

	expected := Outer{
		Flattened: Inner{
			Left:  "flattened_left",
			Right: "flattened_right",
		},
		Inner: Inner{
			Left:  "flattened_left",
			Right: "flattened_right",
		},
	}
	found, err := deserializer.DeserializeKVList(data)
	assert.NilError(t, err)

	assert.DeepEqual(t, *found, expected)
}

// KVListDeserializer works with arrays with non-primitive types with a primitive Kind.
func TestKVDeserializeUnderlyingPrimitiveSlices(t *testing.T) {
	type TestType string
	type TestStruct struct {
		TestField []TestType
	}

	deserializer, err := deserialize.MakeKVListDeserializer[TestStruct](deserialize.QueryOptions(""))
	assert.NilError(t, err)

	sample := TestStruct{
		TestField: []TestType{"abc"},
	}

	kvlist := make(map[string][]string)
	kvlist["TestField"] = []string{"abc"}

	deserialized, err := deserializer.DeserializeKVList(kvlist)
	assert.NilError(t, err)
	assert.DeepEqual(t, *deserialized, sample)
}

// --- Testing side-effects during validation.

type TestValidateModifyStruct struct {
	A string
}

func (t *TestValidateModifyStruct) Validate() error {
	t.A = strings.ToLower(t.A)
	return nil
}

func TestValidateModify(t *testing.T) {
	kvlist := make(map[string][]string)
	kvlist["A"] = []string{"ABC"}

	deserializer, err := deserialize.MakeKVListDeserializer[TestValidateModifyStruct](deserialize.JSONOptions(""))
	assert.NilError(t, err)
	deserialized, err := deserializer.DeserializeKVList(kvlist)
	assert.NilError(t, err)
	fmt.Println(*deserialized)
	assert.Equal(t, deserialized.A, "abc")
}
