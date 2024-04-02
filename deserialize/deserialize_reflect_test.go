package deserialize_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/pasqal-io/godasse/deserialize"
	jsonPkg "github.com/pasqal-io/godasse/deserialize/json"
	"gotest.tools/v3/assert"
)

func twoWaysReflect[Input any, Output any](t *testing.T, sample Input) (*Output, error) {
	var placeholderOutput Output
	typeOutput := reflect.TypeOf(placeholderOutput)
	deserializer, err := deserialize.MakeMapDeserializerFromReflect(deserialize.Options{
		Unmarshaler: jsonPkg.Driver{},
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
	deserialized, err := deserializer.DeserializeDict(dict)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	typed, ok := (*deserialized).(Output)
	if !ok {
		return nil, fmt.Errorf("invalid type after deserailization: expected %s, got %s",
			typeOutput.Name(),
			reflect.TypeOf(deserialized).Elem().Name())
	}
	return &typed, nil
}

func TestReflectDeserializer(t *testing.T) {
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
