package testutils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"testing"
)

// Fail if two values are different.
//
// Does not stop the test.
func AssertEqual[T comparable](t *testing.T, actual, expected T, explanation string) {
	t.Helper()
	if expected != actual {
		t.Errorf("got: %+v; want: %+v (%s)", actual, expected, explanation)
		if reflect.ValueOf(expected).Kind() == reflect.Pointer {
			t.Error("Warning: you're comparing two pointers -- pointers are only equal if they point to the same physical object")
		}
	}
}
func AssertEqualArrays[T comparable](t *testing.T, actual, expected []T, explanation string) {
	t.Helper()
	AssertEqual(t, len(actual), len(expected), fmt.Sprintf("%s - invalid length", explanation))
	for i := 0; i < len(actual); i++ {
		AssertEqual(t, actual[i], expected[i], fmt.Sprintf("%s - invalid item %d", explanation, i))
	}
}

func AssertRegexp(t *testing.T, actual string, pattern regexp.Regexp, explanation string) {
	t.Helper()
	if pattern.FindStringIndex(actual) != nil {
		return
	}
	t.Errorf("got: %+v; expected: %+v (%s)", actual, pattern, explanation)
}

func Unmarshal[T any](t *testing.T, payload []byte) (*T, error) {
	t.Helper()

	// Case 1: Can Payload be unmarshalled to T?
	result := new(T)
	errT := json.Unmarshal(payload, &result)
	if errT == nil {
		return result, nil
	}

	// Case 2: Can Payload can be unmarshalled to any kind of JSON?
	debug := make(map[string]interface{})
	errJSON := json.Unmarshal(payload, &debug)
	if errJSON == nil {
		return nil, fmt.Errorf("payload is valid JSON but not in expected format, got: %+v\n\t%w", debug, errT)
	}

	// Otherwise, return the string.
	//
	// Now, by definition, not all instances of `[]byte` are valid strings, regardless of the encoding.
	// So I'm going to assumem that this will panic if the string is invalid.
	return nil, fmt.Errorf("payload is invalid JSON, got %s\n\t%w", string(payload), errT)
}
