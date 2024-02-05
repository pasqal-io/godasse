package validation_test

import (
	"errors"
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
	if schema == nil {
		return errors.New("nil schema")
	}
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

type ValidatableString string

func (s ValidatableString) Validate() error {
	switch s {
	case "zero":
		fallthrough
	case "one":
		fallthrough
	case "two":
		return nil
	default:
		return fmt.Errorf("Invalid schema kind %s", s)
	}
}

// A trivial test of the validation interface.
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

// Tests for the Validate function.
func TestValidate(t *testing.T) {
	type Pointer[T any] struct {
		Pointer  *T
		Pointer2 *T
		Pointer3 *int
	}
	type Struct[T any] struct {
		Struct T
		Int    int
		Uint   uint
		String string
	}
	type Slice[T any] struct {
		Slice []T
	}
	type Array[T any] struct {
		Array [1]T
	}
	type Interface struct {
		Interface any // But really, an ExampleValidator.
	}
	type MapKey[T comparable] struct {
		MapKey map[T]string
	}
	type MapValue[T any] struct {
		MapValue map[string]T
	}

	// The following values are valid.
	err := validation.Validate(&ExampleValidator{Kind: "one"}) // nolint:exhaustruct
	assert.NilError(t, err)

	err = validation.Validate(&Pointer[ExampleValidator]{ // nolint:exhaustruct
		Pointer:  &ExampleValidator{Kind: "one"}, // nolint:exhaustruct
		Pointer2: &ExampleValidator{Kind: "one"}, // nolint:exhaustruct
	})
	assert.NilError(t, err)

	err = validation.Validate(&Struct[ExampleValidator]{ // nolint:exhaustruct
		Struct: ExampleValidator{Kind: "one"}, // nolint:exhaustruct
	})
	assert.NilError(t, err)

	err = validation.Validate(&Slice[ExampleValidator]{ // nolint:exhaustruct
		Slice: []ExampleValidator{{Kind: "one"}}, // nolint:exhaustruct
	})
	assert.NilError(t, err)

	err = validation.Validate(&Array[ExampleValidator]{ // nolint:exhaustruct
		Array: [1]ExampleValidator{{Kind: "one"}}, // nolint:exhaustruct
	})
	assert.NilError(t, err)

	err = validation.Validate(&Interface{ // nolint:exhaustruct
		Interface: ExampleValidator{Kind: "one"}, // nolint:exhaustruct
	})
	assert.NilError(t, err)

	err = validation.Validate(&MapKey[ExampleValidator]{ // nolint:exhaustruct
		MapKey: map[ExampleValidator]string{{Kind: "one"}: "turee"}, // nolint:exhaustruct
	})
	assert.NilError(t, err)

	err = validation.Validate(&MapValue[ExampleValidator]{ // nolint:exhaustruct
		MapValue: map[string]ExampleValidator{"turee": {Kind: "one"}}, // nolint:exhaustruct
	})
	assert.NilError(t, err)

	one := ValidatableString("one")
	err = validation.Validate[ValidatableString](&one) // nolint:exhaustruct
	assert.NilError(t, err)

	err = validation.Validate(&Pointer[ValidatableString]{ // nolint:exhaustruct
		Pointer:  &one, // nolint:exhaustruct
		Pointer2: &one, // nolint:exhaustruct
	})
	assert.NilError(t, err)

	err = validation.Validate(&Struct[ValidatableString]{ // nolint:exhaustruct
		Struct: ValidatableString("one"), // nolint:exhaustruct
	})
	assert.NilError(t, err)

	err = validation.Validate(&Slice[ValidatableString]{ // nolint:exhaustruct
		Slice: []ValidatableString{one}, // nolint:exhaustruct
	})
	assert.NilError(t, err)

	err = validation.Validate(&Array[ValidatableString]{ // nolint:exhaustruct
		Array: [1]ValidatableString{one}, // nolint:exhaustruct
	})
	assert.NilError(t, err)

	err = validation.Validate(&Interface{ // nolint:exhaustruct
		Interface: one, // nolint:exhaustruct
	})
	assert.NilError(t, err)

	err = validation.Validate(&MapKey[ValidatableString]{ // nolint:exhaustruct
		MapKey: map[ValidatableString]string{one: "turee"}, // nolint:exhaustruct
	})
	assert.NilError(t, err)

	err = validation.Validate(&MapValue[ValidatableString]{ // nolint:exhaustruct
		MapValue: map[string]ValidatableString{"turee": one}, // nolint:exhaustruct
	})
	assert.NilError(t, err)

	// The following values are invalid.
	validError := validation.Error{}
	err = validation.Validate(&ExampleValidator{Kind: "turee"}) // nolint:exhaustruct
	assert.ErrorContains(t, err, "Invalid schema kind turee")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}

	err = validation.Validate(&Pointer[ExampleValidator]{ // nolint:exhaustruct
		Pointer: &ExampleValidator{Kind: "turee"}, // nolint:exhaustruct
	})
	assert.ErrorContains(t, err, "Invalid schema kind turee")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}

	err = validation.Validate(&Pointer[ExampleValidator]{ // nolint:exhaustruct
		Pointer: &ExampleValidator{Kind: "one"}, // nolint:exhaustruct
	})
	assert.ErrorContains(t, err, "nil schema")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}

	err = validation.Validate(&Struct[ExampleValidator]{ // nolint:exhaustruct
		Struct: ExampleValidator{Kind: "turee"}, // nolint:exhaustruct
	})
	assert.ErrorContains(t, err, "Invalid schema kind turee")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}

	err = validation.Validate(&Slice[ExampleValidator]{ // nolint:exhaustruct
		Slice: []ExampleValidator{{Kind: "turee"}}, // nolint:exhaustruct
	})
	assert.ErrorContains(t, err, "Invalid schema kind turee")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}

	err = validation.Validate(&Array[ExampleValidator]{ // nolint:exhaustruct
		Array: [1]ExampleValidator{{Kind: "turee"}}, // nolint:exhaustruct
	})
	assert.ErrorContains(t, err, "Invalid schema kind turee")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}

	err = validation.Validate(&Interface{ // nolint:exhaustruct
		Interface: ExampleValidator{Kind: "turee"}, // nolint:exhaustruct
	})
	assert.ErrorContains(t, err, "Invalid schema kind turee")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}

	err = validation.Validate(&MapKey[ExampleValidator]{ // nolint:exhaustruct
		MapKey: map[ExampleValidator]string{{Kind: "turee"}: "one"}, // nolint:exhaustruct
	})
	assert.ErrorContains(t, err, "Invalid schema kind turee")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}

	err = validation.Validate(&MapValue[ExampleValidator]{ // nolint:exhaustruct
		MapValue: map[string]ExampleValidator{"one": {Kind: "turee"}}, // nolint:exhaustruct
	})
	assert.ErrorContains(t, err, "Invalid schema kind turee")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}

	turee := ValidatableString("turee")
	err = validation.Validate[ValidatableString](&turee) // nolint:exhaustruct
	assert.ErrorContains(t, err, "Invalid schema kind turee")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}

	err = validation.Validate(&Pointer[ValidatableString]{ // nolint:exhaustruct
		Pointer:  &turee, // nolint:exhaustruct
		Pointer2: &turee, // nolint:exhaustruct
	})
	assert.ErrorContains(t, err, "Invalid schema kind turee")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}

	err = validation.Validate(&Struct[ValidatableString]{ // nolint:exhaustruct
		Struct: turee, // nolint:exhaustruct
	})
	assert.ErrorContains(t, err, "Invalid schema kind turee")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}

	err = validation.Validate(&Slice[ValidatableString]{ // nolint:exhaustruct
		Slice: []ValidatableString{turee}, // nolint:exhaustruct
	})
	assert.ErrorContains(t, err, "Invalid schema kind turee")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}

	err = validation.Validate(&Array[ValidatableString]{ // nolint:exhaustruct
		Array: [1]ValidatableString{turee}, // nolint:exhaustruct
	})
	assert.ErrorContains(t, err, "Invalid schema kind turee")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}

	err = validation.Validate(&Interface{ // nolint:exhaustruct
		Interface: turee, // nolint:exhaustruct
	})
	assert.ErrorContains(t, err, "Invalid schema kind turee")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}

	err = validation.Validate(&MapKey[ValidatableString]{ // nolint:exhaustruct
		MapKey: map[ValidatableString]string{turee: "turee"}, // nolint:exhaustruct
	})
	assert.ErrorContains(t, err, "Invalid schema kind turee")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}

	err = validation.Validate(&MapValue[ValidatableString]{ // nolint:exhaustruct
		MapValue: map[string]ValidatableString{"turee": turee}, // nolint:exhaustruct
	})
	assert.ErrorContains(t, err, "Invalid schema kind turee")
	if ok := errors.As(err, &validError); !ok {
		t.Fatal("invalid error, expected a validation.Error, got", err)
	}
}
