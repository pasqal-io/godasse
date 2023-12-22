package validation_test

import (
	"fmt"
	"testing"

	"github.com/pasqal-io/godasse/validation"
	"gotest.tools/v3/assert"
)

type ExamplePayload struct {
	left   string
	right  string
	middle string
}

type ExampleInitializer struct {
	payload *ExamplePayload
}

func (schema *ExampleInitializer) Initialize() error {
	schema.payload = &ExamplePayload{ //nolint:exhaustruct
		left:  "left",
		right: "right",
	}
	return nil
}

var _ validation.Initializer = &ExampleInitializer{} //nolint:exhaustruct

// A trivial test of initialization.
//
// See the tests for deserialize for more advanced checks.
func TestInitialization(t *testing.T) {
	result := ExampleInitializer{} //nolint:exhaustruct
	err := result.Initialize()
	if err != nil {
		t.Error(err)
		return
	}
	assert.Equal(t, result.payload.left, "left", "Field left should have been set")
	assert.Equal(t, result.payload.right, "right", "Field right should have been set")
	assert.Equal(t, result.payload.middle, "", "Field middle should have been zeroed")
}

type ExampleValidator struct {
	Kind      string `json:"kind"`    // Public field, will be deserialized.
	kindIndex uint   `initialized:""` // Private field, will e initialized by the validation step.
}

func (schema *ExampleValidator) Validate() error {
	switch schema.Kind {
	case "zero":
		schema.kindIndex = 0
	case "one":
		schema.kindIndex = 1
	case "two":
		schema.kindIndex = 2
	default:
		return fmt.Errorf("Invalid schema kind %s", schema.Kind)
	}
	// Success.
	return nil
}

var _ validation.Validator = &ExampleValidator{} //nolint:exhaustruct

// A trivial test of validation.
//
// See the tests for deserialize for more advanced checks.
func TestValidation(t *testing.T) {
	// This should pass validation.
	good := ExampleValidator{ //nolint:exhaustruct
		Kind: "one",
	}
	err := good.Validate()
	if err != nil {
		t.Error(err)
		return
	}
	assert.Equal(t, good.Kind, "one", "Field Kind should have been left unchanged")
	assert.Equal(t, good.kindIndex, uint(1), "Field kindIndex should have been set")

	// This shouldn't.
	bad := ExampleValidator{ //nolint:exhaustruct
		Kind: "three",
	}
	err = bad.Validate()
	assert.Equal(t, err.Error(), "Invalid schema kind three", "Validation should reject")
}
