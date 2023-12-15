package initialized

// A witness type used to detect structs that are not initialized.
//
// By design, in Go, there are several ways to create a struct without going
// through its constructor. In fact, in many cases, it's actually easier to
// create a struct without going through its constructor than to go through
// the constructor:
//
//		foo := new(T)
//
//		foo T{}
//
//	 or foo T{/* fields*/} // outside of constructor
//
// This makes it all too easy to have a value that pretends to have type `T`
// both at compile-time and at run-time but does not actually offer any of
// the guarantees attached to `T`.
//
// IsInialized is a mechanism to alleviate the cognitive load of tracking
// which structs are initialized.
//
// Operation manual:
// - add a field `witness IsInitialized` in your struct
// - call `assert_init.Make()` from your constructor
// - call `self.witness.Assert()` whenever you access data from your struct.
//
// Result:
//   - this will cause a panic if your struct was not properly initialized,
//     e.g. if we're accidentally using a struct created with missing fields,
//     or constructed with `new` instead of the constructor.
//
// Note: when deserialized, `IsInitialized` adopts the proper default of
// initializing itself.
type IsInitialized struct {
	isInitialized bool `default:"true"`
}

// Create a `IsInitialized`.
func Make() IsInitialized {
	return IsInitialized{
		isInitialized: true,
	}
}

// Assert that this `IsInitialized` has been initialized, i.e. if it was created
// by calling `initialized.Make()`.
//
// Panics if it wasn't, i.e. if the container struct was created by calling
//
//		foo := new(T)
//
//		foo T{}
//
//	 or foo T{/* fields*/} // outside of constructor
func (witness IsInitialized) Assert() {
	if !witness.isInitialized {
		panic("Struct was not initialized")
	}
}
