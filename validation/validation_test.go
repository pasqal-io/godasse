package validation_test

import (
	"fmt"
	"testing"

	"github.com/pasqal-io/godasse/assertions/testutils"
	"github.com/pasqal-io/godasse/validation"
)

type ExamplePayload struct {
	left   string
	right  string
	middle string
}

type ExampleCanInitialize struct {
	payload *ExamplePayload
}

func (schema *ExampleCanInitialize) Initialize() error {
	schema.payload = &ExamplePayload{ //nolint:exhaustruct
		left:  "left",
		right: "right",
	}
	return nil
}

var _ validation.CanInitialize = &ExampleCanInitialize{} //nolint:exhaustruct

// A trivial test of initialization.
//
// See the tests for deserialize for more advanced checks.
func TestInitialization(t *testing.T) {
	result := ExampleCanInitialize{} //nolint:exhaustruct
	err := result.Initialize()
	if err != nil {
		t.Error(err)
		return
	}
	testutils.AssertEqual(t, result.payload.left, "left", "Field left should have been set")
	testutils.AssertEqual(t, result.payload.right, "right", "Field right should have been set")
	testutils.AssertEqual(t, result.payload.middle, "", "Field middle should have been zeroed")
}

type ExampleCanValidate struct {
	Kind      string `json:"kind"`    // Public field, will be deserialized.
	kindIndex uint   `initialized:""` // Private field, will e initialized by the validation step.
}

func (schema *ExampleCanValidate) Validate() error {
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

var _ validation.CanValidate = &ExampleCanValidate{} //nolint:exhaustruct

// A trivial test of validation.
//
// See the tests for deserialize for more advanced checks.
func TestValidation(t *testing.T) {
	// This should pass validation.
	good := ExampleCanValidate{
		Kind: "one",
	} //nolint:exhaustruct
	err := good.Validate()
	if err != nil {
		t.Error(err)
		return
	}
	testutils.AssertEqual(t, good.Kind, "one", "Field Kind should have been left unchanged")
	testutils.AssertEqual(t, good.kindIndex, 1, "Field kindIndex should have been set")

	// This shouldn't.
	bad := ExampleCanValidate{
		Kind: "three",
	} //nolint:exhaustruct
	err = bad.Validate()
	testutils.AssertEqual(t, err.Error(), "Invalid schema kind three", "Validation should reject")
}
