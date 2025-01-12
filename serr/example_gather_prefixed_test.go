package serr_test

import (
	"errors"
	"fmt"

	"github.com/ngicks/go-common/serr"
)

// ExampleGatherPrefixed shows main use case of this module.
//
// GatherPrefixed concatenates multiple errors with prefix added to each error.
// GatherPrefixed returns non-nil error only when at least a single non nil error is given.
// It also keeps nil error in the returned error for better readability and consistency.
func ExampleGatherPrefixed() {
	err := serr.GatherPrefixed(
		serr.ToPairs(
			map[string]error{
				"0=": nil,
				"1=": nil,
			},
			nil,
		),
	)
	fmt.Printf("nil if every error is nil = %v\n", err)

	err = serr.GatherPrefixed(
		serr.ToPairs(
			map[string]error{
				"0=": errors.New("foo"),
				"1=": nil,
				"2=": errors.New("baz"),
			},
			nil,
		),
	)
	fmt.Printf("at least a non nil error = %v\n", err)
	fmt.Printf("verbs propagates = %#v\n", err)

	// Output:
	//
	// nil if every error is nil = <nil>
	// at least a non nil error = 0=foo, 1=<nil>, 2=baz
	// verbs propagates = 0=&errors.errorString{s:"foo"}, 1=<nil>, 2=&errors.errorString{s:"baz"}
}
