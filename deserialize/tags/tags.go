package tags

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/pasqal-io/godasse/assertions/initialized"
)

// A representation of the tags for a given field.
type Tags struct {
	tags    map[string][]string
	witness initialized.IsInitialized
}

func Empty() Tags {
	return Tags{
		tags:    make(map[string][]string),
		witness: initialized.Make(),
	}
}

// Parse the tag associated to a struct field, according to the specs
// of Go tags.
func Parse(tag reflect.StructTag) (Tags, error) {
	tags := make(map[string][]string)
	// Copied and pasted from Go's type.go.
	for tag != "" {
		// Skip leading space.
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// Scan to colon. A space, a quote or a control character is a syntax error.
		// Strictly speaking, control chars include the range [0x7f, 0x9f], not just
		// [0x00, 0x1f], but in practice, we ignore the multi-byte control characters
		// as it is simpler to inspect the tag's bytes than the tag's runes.
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			// Give up on parsing.
			break
		}
		name := string(tag[:i])
		if name == "" {
			return Tags{}, errors.New("invalid tag with empty name")
		}
		if _, exists := tags[name]; exists {
			return Tags{}, fmt.Errorf("invalid tag, name %s should only be defined once", name)
		}

		tag = tag[i+1:]

		// Scan quoted string to find value.
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]

		list, err := strconv.Unquote(qvalue)
		if err != nil {
			return Tags{}, fmt.Errorf("ill-formed tag %s:\n\t * %w", name, err)
		}

		switch name {
		case "default":
			fallthrough
		case "orMethod":
			// don't pre-process
			tags[name] = []string{list}
		default:
			split := strings.Split(list, ",")
			trimmed := make([]string, 0)
			for _, s := range split {
				t := strings.Trim(s, " ")
				if t != "" {
					trimmed = append(trimmed, t)
				}
			}
			// Make sure that we always have at least an empty string.
			if len(trimmed) == 0 {
				trimmed = append(trimmed, "")
			}
			tags[name] = trimmed
		}
	}
	return Tags{
		tags:    tags,
		witness: initialized.Make(),
	}, nil
}

// Return the a default value that may be used to initialize a
// field if no value is provided.
//
// This is tag `default`. Conflicts with `orMethod`.
func (tags Tags) Default() *string {
	tags.witness.Assert()
	result, ok := tags.tags["default"]
	if !ok || len(result) == 0 {
		return nil
	}

	return &result[0]
}

// Return the name of a method that may be used to initialize
// a field if no value is provided.
//
// This is tag `orMethod`. Conflicts with `default`.
func (tags Tags) MethodName() *string {
	tags.witness.Assert()
	result, ok := tags.tags["orMethod"]
	if !ok || len(result) == 0 {
		return nil
	}
	return &result[0]
}

// Return the public field name for a field.
//
// e.g. for json, if there's a tag `json:"foo"`, this means
// that the field should be imported as `foo`.
func (tags Tags) PublicFieldName(key string) *string {
	tags.witness.Assert()
	result, ok := tags.tags[key]
	if !ok || len(result) == 0 {
		return nil
	}
	return &result[0]
}

// Return `true` if this field should be considered pre-initialized
// (i.e. the parser should not complain of any fields immediately within
// that field), `false` otherwise.
//
// It generally makes sense only for structs (slices or pointers thereof).
//
// This is tag `initialized`.
func (tags Tags) IsPreinitialized() bool {
	tags.witness.Assert()
	_, ok := tags.tags["initialized"]
	return ok
}

// Return `true` if this field is marked as `flatten`, e.g.
//
//	type Flattening struct {
//	    A string
//	    B struct {
//	        C string
//	        D string
//	    } // `flatten:""`
//	}
//
// should deserialized from the following JSON
//
//	{
//	   "A": "aaaaa",
//	   // no field B
//	   "C": "ccccc",
//	   "D": "ddddd"
//	}
func (tags Tags) IsFlattened() bool {
	tags.witness.Assert()
	_, ok := tags.tags["flatten"]
	return ok
}

// Lookup a key.
func (tags Tags) Lookup(key string) ([]string, bool) {
	tags.witness.Assert()
	result, ok := tags.tags[key]
	return result, ok
}
