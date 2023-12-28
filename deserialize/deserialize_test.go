//nolint:exhaustruct
package deserialize_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/pasqal-io/godasse/deserialize"
	jsonPkg "github.com/pasqal-io/godasse/deserialize/json"
	"github.com/pasqal-io/godasse/deserialize/kvlist"
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
	SomeArray []string
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
	return fmt.Errorf("Invalid email")
}

var _ validation.Validator = &ValidatedStruct{} // Type assertion.

func twoWaysGeneric[Input any, Output any](t *testing.T, sample Input) (*Output, error) {
	deserializer, err := deserialize.MakeMapDeserializer[Output](deserialize.Options{
		Unmarshaler: jsonPkg.Driver{},
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
	array := make([]string, 3)
	array[0] = "one"
	array[1] = "two"
	array[2] = "three"
	before := SimpleArrayStruct{
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

func TestDeserializeArrayOfStruct(t *testing.T) {
	array := make([]SimpleStruct, 3)
	array[0] = SimpleStruct{"one"}
	array[1] = SimpleStruct{"two"}
	array[2] = SimpleStruct{"three"}
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
	assert.Equal(t, err.Error(), "deserialized value ValidatedStruct did not pass validation\n\t * Invalid email", "Validation should have caught the error")
}

func TestValidationFailureFieldField(t *testing.T) {
	before := Pair[int, ValidatedStruct]{
		Left: 0,
		Right: ValidatedStruct{
			SomeEmail: "someone+example.com",
		},
	}
	_, err := twoWays(t, before)
	assert.Equal(t, err.Error(), "deserialized value Pair[int,ValidatedStruct].right did not pass validation\n\t * Invalid email", "Validation should have caught the error")
}

func TestValidationFailureArray(t *testing.T) {
	array := make([]ValidatedStruct, 1)
	array[0] = ValidatedStruct{
		SomeEmail: "someone+example.com",
	}
	before := Array[ValidatedStruct]{
		Data: array,
	}
	_, err := twoWays(t, before)
	assert.Equal(t, err.Error(), "error while deserializing Array[ValidatedStruct].Data[0]:\n\t * deserialized value Array[ValidatedStruct].Data[] did not pass validation\n\t * Invalid email", "Validation should have caught the error")
}

func TestKVListDoesNotSupportNesting(t *testing.T) {
	options := deserialize.QueryOptions("") //nolint:exhaustruct
	_, err := deserialize.MakeKVListDeserializer[PrimitiveTypesStruct](options)
	assert.NilError(t, err, "KVList parsing supports simple structurs")

	_, err = deserialize.MakeKVListDeserializer[SimpleArrayStruct](options)
	assert.Equal(t, err.Error(), "this type of extractor does not support arrays/slices", "KVList parsing does not support nesting")

	_, err = deserialize.MakeKVListDeserializer[Pair[int, Pair[int, int]]](options)
	assert.Equal(t, err.Error(), "this type of extractor does not support nested structs", "KVList parsing does not support nesting")
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

	assert.Equal(t, err.Error(), "could not generate a deserializer for PairWithDefaults[PairWithDefaults[EmptyStruct,int]·5,int].Left with type PairWithDefaults[EmptyStruct,int]:\n\t * could not generate a deserializer for PairWithDefaults[PairWithDefaults[EmptyStruct,int]·5,int].Left.Right with type int:\n\t * cannot parse default value at PairWithDefaults[PairWithDefaults[EmptyStruct,int]·5,int].Left.Right\n\t * strconv.Atoi: parsing \"{}\": invalid syntax", "MakeMapDeserializer should have detected an error")
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
		Data []EmptyStruct `default:"[]"`
	}
	result, err := twoWaysGeneric[EmptyStruct, ArrayWithDefault](t, EmptyStruct{})
	assert.NilError(t, err)
	assert.Equal(t, len(result.Data), 0, "Deserialization should have inserted default values")
}

// Test that default values are parsed.
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

type SimpleStructWithOrMethodSuccess struct {
	SomeString string `orMethod:"MakeString"`
}

func (SimpleStructWithOrMethodSuccess) MakeString() (string, error) {
	return "Test value constructed with a method", nil
}

// Test that an `orMethod` will be called if no value is provided (success case).
func TestOrMethodSuccess(t *testing.T) {
	expected := SimpleStructWithOrMethodSuccess{
		SomeString: "Test value constructed with a method",
	}
	result, err := twoWaysGeneric[EmptyStruct, SimpleStructWithOrMethodSuccess](t, EmptyStruct{})
	assert.NilError(t, err)
	assert.Equal(t, *result, expected, "The method should have been called to inject a value")
}

type SimpleStructWithOrMethodError struct {
	SomeString string `orMethod:"MakeString"`
}

func (SimpleStructWithOrMethodError) MakeString() (string, error) {
	return "Test value constructed with a method", fmt.Errorf("This is an error from SimpleStructWithOrMethodError")
}

// Test that an `orMethod` will be called if no value is provided (error case).
func TestOrMethodError(t *testing.T) {
	_, err := twoWaysGeneric[EmptyStruct, SimpleStructWithOrMethodError](t, EmptyStruct{})
	assert.Equal(t, err.Error(), "error in optional value at SimpleStructWithOrMethodError.SomeString\n\t * This is an error from SimpleStructWithOrMethodError", "The method should have been called to inject a value")
}

type SimpleStructWithOrMethodBadName struct {
	SomeString string `orMethod:"IDoNotExist"`
}
type SimpleStructWithOrMethodBadArgs struct {
	SomeString string `orMethod:"BadArgs"`
}

func (SimpleStructWithOrMethodBadArgs) BadArgs(string) (string, error) {
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

func (SimpleStructWithOrMethodBadOut1) BadOut2() (string, string) {
	return "", ""
}

// Test error cases for `onMethod` setup.
func TestOrMethodBadSetup(t *testing.T) {
	_, err := deserialize.MakeMapDeserializer[SimpleStructWithOrMethodBadName](deserialize.JSONOptions("")) //nolint:exhaustruct
	assert.Equal(t, err.Error(), "could not generate a deserializer for SimpleStructWithOrMethodBadName.SomeString with type string:\n\t * at SimpleStructWithOrMethodBadName.SomeString, failed to setup `orMethod`\n\t * method IDoNotExist provided with `orMethod` doesn't seem to exist - note that the method must be public", "We should fail early if the orMethod doesn't exist")

	_, err = deserialize.MakeMapDeserializer[SimpleStructWithOrMethodBadArgs](deserialize.JSONOptions(""))
	assert.Equal(t, err.Error(), "could not generate a deserializer for SimpleStructWithOrMethodBadArgs.SomeString with type string:\n\t * at SimpleStructWithOrMethodBadArgs.SomeString, failed to setup `orMethod`\n\t * the method provided with `orMethod` MUST take no argument but takes 1 arguments", "We should fail early if orMethod args are incorrect")

	_, err = deserialize.MakeMapDeserializer[SimpleStructWithOrMethodBadOut1](deserialize.JSONOptions(""))
	assert.Equal(t, err.Error(), "could not generate a deserializer for SimpleStructWithOrMethodBadOut1.SomeInt with type int:\n\t * at SimpleStructWithOrMethodBadOut1.SomeInt, failed to setup `orMethod`\n\t * the method provided with `orMethod` MUST return (int, error) but it returns (string, _) which is not convertible to `int`", "We should fail early if first result is incorrect")

	_, err = deserialize.MakeMapDeserializer[SimpleStructWithOrMethodBadOut2](deserialize.JSONOptions(""))
	assert.Equal(t, err.Error(), "could not generate a deserializer for SimpleStructWithOrMethodBadOut2.SomeString with type string:\n\t * at SimpleStructWithOrMethodBadOut2.SomeString, failed to setup `orMethod`\n\t * method BadOut2 provided with `orMethod` doesn't seem to exist - note that the method must be public", "We should fail early if second result is incorrect")
}

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
	result := make([]SimpleStruct, 2)
	result[0] = SimpleStruct{
		SomeString: "I have been made 1",
	}
	result[1] = SimpleStruct{
		SomeString: "I have been made 2",
	}
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
	return fmt.Errorf("Test error")
}

var _ validation.Initializer = &StructInitializerFaulty{} // Type assertion.

// Test that we're correctly running pre-initialization at toplevel.
func TestInitializerFaulty(t *testing.T) {
	sample := EmptyStruct{}
	_, err := twoWaysGeneric[EmptyStruct, StructInitializerFaulty](t, sample)
	assert.Equal(t, err.Error(), "at StructInitializerFaulty, encountered an error while initializing optional fields:\n\t * Test error", "Initializer should have initialized our structure")

	asCustom := deserialize.CustomDeserializerError{}
	ok := errors.As(err, &asCustom)
	assert.Equal(t, ok, true, "the error should be a CustomDeserializerError")
}

type StructUnmarshal struct {
	hidden string
}

func (su *StructUnmarshal) UnmarshalJSON(source []byte) error {
	if len(source) == 0 {
		return fmt.Errorf("Test error: this slice is too short")
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
