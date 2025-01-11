package serr

import (
	"fmt"
	"io"
	"strings"
)

var _ error = (*multiError)(nil)
var _ fmt.Formatter = (*multiError)(nil)

type multiError struct{ errs []error }

// NewMultiError wraps errors into single error, ignoring nil values in errs.
//
// If all errors are nil or len(errs) == 0, NewMultiError returns nil.
//
// errs is retained by returned error.
// Callers should not mutate errs after NewMultiErrorChecked returns.
func NewMultiError(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	var i int
	for i = 0; i < len(errs); i++ {
		if errs[i] == nil {
			break
		}
	}
	if i == len(errs) {
		return NewMultiErrorUnchecked(errs)
	}
	var filtered []error
	for _, err := range errs {
		if err != nil {
			filtered = append(filtered, err)
		}
	}

	if len(filtered) == 0 {
		return nil
	}

	return &multiError{errs: filtered}
}

// NewMultiErrorChecked wraps errors into single error if and only if errs contains at least one non nil error.
// It also preserves nil errors in errs for better printing.
// This is useful when an error itself does not contain information
// to pin-point how and why error is caused other than just index within error slice.
//
// NewMultiErrorChecked returns nil if len(errs) == 0 or all errors are nil.
//
// errs is retained by returned error.
// Callers should not mutate errs after NewMultiErrorChecked returns.
func NewMultiErrorChecked(errs []error) error {
	if len(errs) == 0 {
		return nil
	}

	containsNonNil := false
	for _, e := range errs {
		if e != nil {
			containsNonNil = true
			break
		}
	}
	if !containsNonNil {
		return nil
	}
	return NewMultiErrorUnchecked(errs)
}

// NewMultiErrorUnchecked wraps errors into single error.
// As suffix "unchecked" implies it does not do any filtering for errs.
// The returned error is always non nil even if all errors are nil or len(errs) == 0.
//
// errs is retained by returned error.
// Callers should not mutate errs after NewMultiErrorChecked returns.
func NewMultiErrorUnchecked(errs []error) error {
	return &multiError{errs: errs}
}

func (me *multiError) Unwrap() []error {
	return me.errs
}

func (me *multiError) format(w io.Writer, fmtStr string) {
	_, _ = io.WriteString(w, "MultiError: ")
	for i, err := range me.errs {
		if i > 0 {
			_, _ = w.Write([]byte(`, `))
		}
		_, _ = fmt.Fprintf(w, fmtStr, err)
	}
}

func (me *multiError) Error() string {
	var s strings.Builder
	me.format(&s, "%s")
	return s.String()
}

// Format implements fmt.Formatter.
//
// Format propagates given flags, width, precision and verb into each error.
// Then it concatenates each result with ", " suffix.
//
// Without Format, me is less readable when printed through fmt.*printf family functions.
// e.g. Format produces lines like
// (%+v) MultiError: errors, exampleErr: Foo=foo Bar=bar Baz=baz
// (%#v) MultiError: &errors.errorString{s:"errors"}, &mymodule.exampleErr{Foo:"foo", Bar:"bar", Baz:"baz"}
// instead of (w/o Format)
// (%+v) stream.multiError{(*errors.errorString)(0xc00002c300), (*mymodule.exampleErr)(0xc000102630)}
// (%#v) [824633901824 824634779184]
func (me *multiError) Format(state fmt.State, verb rune) {
	me.format(state, fmt.FormatString(state, verb))
}
