package internal

import "github.com/pasqal-io/godasse/deserialize/shared"

// A trivial implementation of shared.Value
// containing nothing.
type EmptyValue struct {
}

func (empty EmptyValue) AsDict() (shared.Dict, bool) {
	return EmptyDict{}, true
}
func (empty EmptyValue) AsSlice() ([]shared.Value, bool) {
	return make([]shared.Value, 0), false
}
func (empty EmptyValue) Interface() any {
	return nil
}

var _ shared.Value = EmptyValue{}

// A trivial implementation of shared.Dict
// containing nothing.
type EmptyDict struct {
}

func (empty EmptyDict) Lookup(key string) (shared.Value, bool) {
	return EmptyValue{}, false
}
func (empty EmptyDict) AsValue() shared.Value {
	return EmptyValue{}
}

var _ shared.Dict = EmptyDict{}
