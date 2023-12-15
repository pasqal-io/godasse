// Mechanisms to deal with initialization and validation of values.
//
// These interfaces are primarily designed to be implemented by
// deserialization schemas.
package validation

// A type that supports initialization.
//
// Our deserialization library automatically runs any call to `Initialize()`
// at every depth of the tree, **before** building the node.
//
// Important: We expect `CanValidate` to be implemented on **pointers**,
// rather than on structs.
//
// Otherwise, all its operations are performed on a copy of the struct and
// the result is lost immediately.
type CanInitialize interface {
	// Setup the contents of the struct.
	Initialize() error
}

// A type that supports validation.
//
// Our deserialization library automatically runs any call to `Validate()`,
// at every depth of the tree, **after** building the node.
//
// Important: We expect `CanValidate` to be implemented on **pointers**,
// rather than on structs.
//
// This lets `Validate()` perform any necessary changes to the data
// structure. In particular, if necessary, it may be used to populate
// private fields from the contents of public fields.
type CanValidate interface {
	// Confirm that the data is valid.
	//
	// Return an error if it is invalid.
	//
	// If necessary, this method may alter the contents of the struct.
	Validate() error
}
