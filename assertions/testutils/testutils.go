package testutils

import (
	"encoding/json"
	"fmt"
	"testing"
)

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
