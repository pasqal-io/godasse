package deserialize_test

import (
	"encoding/json"
	"reflect"
	"strconv"
	"testing"

	"github.com/pasqal-io/godasse/deserialize"
	jsonPkg "github.com/pasqal-io/godasse/deserialize/json"
	"gotest.tools/v3/assert"
)

func twoWaysReflect[Input any, Output any](t *testing.T, sample Input) (*Output, error) {
	var placeholderOutput Output
	typeOutput := reflect.TypeOf(placeholderOutput)
	deserializer, err := deserialize.MakeMapDeserializerFromReflect(deserialize.Options{
		Unmarshaler: jsonPkg.Driver,
		MainTagName: "json",
		RootPath:    "",
	}, typeOutput)
	if err != nil {
		t.Error(err)
		return nil, err //nolint:wrapcheck
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
	deserialized := new(Output)
	reflectDeserialized := reflect.ValueOf(deserialized).Elem()
	err = deserializer.DeserializeDictTo(dict, &reflectDeserialized)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	return deserialized, nil
}

func TestReflectMapDeserializer(t *testing.T) {
	type Test struct {
		String string
		Int    int
	}
	sample := Test{
		String: "abc",
		Int:    123,
	}
	out, err := twoWaysReflect[Test, Test](t, sample)
	if err != nil {
		t.Fatal(err)
	}
	assert.DeepEqual(t, &sample, out)
}

func TestReflectKVDeserializer(t *testing.T) {
	type Test struct {
		String string
		Int    int
	}
	sample := Test{
		String: "abc",
		Int:    123,
	}
	deserializer, err := deserialize.MakeKVDeserializerFromReflect(deserialize.Options{
		Unmarshaler: jsonPkg.Driver,
		MainTagName: "json",
		RootPath:    "",
	}, reflect.TypeOf(sample))

	assert.NilError(t, err)
	kvList := map[string][]string{}
	kvList["String"] = []string{sample.String}
	kvList["Int"] = []string{strconv.Itoa(sample.Int)}

	deserialized := new(Test)
	reflectDeserialized := reflect.ValueOf(deserialized).Elem()
	err = deserializer.DeserializeKVListTo(kvList, &reflectDeserialized)
	assert.NilError(t, err)
	assert.Equal(t, *deserialized, sample)
}
