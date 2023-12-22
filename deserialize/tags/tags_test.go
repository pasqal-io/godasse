package tags_test

import (
	"reflect"
	"testing"

	"github.com/pasqal-io/godasse/deserialize/tags"
	"gotest.tools/v3/assert"
)

type RandomStruct struct {
	ABC           string  `first:"1,2,3" second:"" third:"abc" fourth:"1,     2,3" fifth:"    abc  " `
	DefaultString string  `default:""`
	DefaultNil    *string `default:"nil"`
	DefaultStruct string  `default:"{}"`
	Repeat        string  `abc:"" abc:""` //lint:ignore SA5008 we're testing for this
	Interesting   string  `default:"abc, def" orMethod:"SomeMethod" renaming:"interesting" initialized:"arbitrary content"`
}

func TestReadTags(t *testing.T) {
	reflectT := reflect.TypeOf(RandomStruct{}) //nolint:exhaustruct
	reflectField, _ := reflectT.FieldByName("ABC")
	parsed, err := tags.Parse(reflectField.Tag)
	if err != nil {
		t.Error("Failed to parse tags ", err)

		return
	}

	first, ok := parsed.Lookup("first")
	if !ok {
		t.Error("Could not find key first")

		return
	}
	assert.DeepEqual(t, first, []string{"1", "2", "3"})

	second, ok := parsed.Lookup("second")
	if !ok {
		t.Error("Could not find key second")

		return
	}
	assert.DeepEqual(t, second, []string{""})

	third, ok := parsed.Lookup("third")
	if !ok {
		t.Error("Could not find key third")

		return
	}
	assert.DeepEqual(t, third, []string{"abc"})

	fourth, ok := parsed.Lookup("fourth")
	if !ok {
		t.Error("Could not find key fourth")

		return
	}
	assert.DeepEqual(t, fourth, []string{"1", "2", "3"})

	fifth, ok := parsed.Lookup("fifth")
	if !ok {
		t.Error("Could not find key fifth")

		return
	}
	assert.DeepEqual(t, fifth, []string{"abc"})

	_, ok = parsed.Lookup("absent")
	if ok {
		t.Error("I should not have found a non-existent key")

		return
	}

	isPreinitialized := parsed.IsPreinitialized()
	if isPreinitialized {
		t.Error("This field is not preinitialized")

		return

	}
}

func TestDefaultString(t *testing.T) {
	reflectT := reflect.TypeOf(RandomStruct{}) //nolint:exhaustruct
	reflectField, _ := reflectT.FieldByName("DefaultString")
	parsed, err := tags.Parse(reflectField.Tag)
	if err != nil {
		t.Error("Failed to parse tags ", err)

		return
	}

	defaultValue := parsed.Default()
	assert.Equal(t, *defaultValue, "", "Should return a default tag")
}

func TestDefaultNil(t *testing.T) {
	reflectT := reflect.TypeOf(RandomStruct{}) //nolint:exhaustruct
	reflectField, _ := reflectT.FieldByName("DefaultNil")
	parsed, err := tags.Parse(reflectField.Tag)
	if err != nil {
		t.Error("Failed to parse tags ", err)

		return
	}

	defaultValue := parsed.Default()
	assert.Equal(t, *defaultValue, "nil", "Should return a default tag")
}

func TestDefaultStruct(t *testing.T) {
	reflectT := reflect.TypeOf(RandomStruct{}) //nolint:exhaustruct
	reflectField, _ := reflectT.FieldByName("DefaultStruct")
	parsed, err := tags.Parse(reflectField.Tag)
	if err != nil {
		t.Error("Failed to parse tags ", err)

		return
	}

	defaultValue := parsed.Default()
	assert.Equal(t, *defaultValue, "{}", "Should return a default tag")
}

// We should fail parsing if the same key appears more than once.
func TestRepeatFails(t *testing.T) {
	reflectT := reflect.TypeOf(RandomStruct{}) //nolint:exhaustruct
	reflectField, _ := reflectT.FieldByName("Repeat")
	_, err := tags.Parse(reflectField.Tag)
	if err == nil {
		t.Error("A key was repeated, we should have failed to parse")

		return
	}

}

// Test meaningful keys.
func TestInteresting(t *testing.T) {
	reflectT := reflect.TypeOf(RandomStruct{}) //nolint:exhaustruct
	reflectField, _ := reflectT.FieldByName("Interesting")
	parsed, err := tags.Parse(reflectField.Tag)
	if err != nil {
		t.Error("Failed to parse tags ", err)

		return
	}

	defaultValue := parsed.Default()
	assert.Equal(t, *defaultValue, "abc, def", "Default value should have remained untrimmed")

	isPreinitialized := parsed.IsPreinitialized()
	assert.Equal(t, isPreinitialized, true, "This field is preinitialized")

	orMethod := parsed.MethodName()
	assert.Equal(t, *orMethod, "SomeMethod", "We should have returned the name of the orMethod")

	publicName := parsed.PublicFieldName("renaming")
	assert.Equal(t, *publicName, "interesting", "We should have returned the correct renaming")
}
