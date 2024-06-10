package serr

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"
)

func TestMultiError(t *testing.T) {
	for _, errs := range [][]error{
		{nil, nil},
		{},
		nil,
	} {
		assertNilInterface(t, NewMultiError(errs))
		assertBool(t, NewMultiErrorUnchecked(errs) != nil, "not NewMultiErrorUnchecked(errs) != nil")
	}

	type testCase struct {
		verb     string
		expected string
	}
	for _, tc := range []testCase{
		{verb: "%s", expected: "MultiError: errors, exampleErr: Foo=foo Bar=bar Baz=baz"},
		{verb: "%v", expected: "MultiError: errors, exampleErr: Foo=foo Bar=bar Baz=baz"},
		{verb: "%+v", expected: "MultiError: errors, exampleErr: Foo=foo Bar=bar Baz=baz"},
		{verb: "%#v", expected: "MultiError: &errors.errorString{s:\"errors\"}, &stream.exampleErr{Foo:\"foo\", Bar:\"bar\", Baz:\"baz\"}"},
		{verb: "%d", expected: "MultiError: &{%!d(string=errors)}, &{%!d(string=foo) %!d(string=bar) %!d(string=baz)}"},
		{verb: "%T", expected: "*stream.multiError"},
		{verb: "%9.3f", expected: "MultiError: &{%!f(string=      err)}, &{%!f(string=      foo) %!f(string=      bar) %!f(string=      baz)}"},
	} {
		tc := tc
		t.Run(tc.verb, func(t *testing.T) {
			e := NewMultiErrorUnchecked([]error{errors.New("errors"), &exampleErr{"foo", "bar", "baz"}})
			formatted := fmt.Sprintf(tc.verb, e)
			assertEq(t, tc.expected, formatted)
		})
	}

	nilMultiErr := NewMultiErrorUnchecked(nil)
	assertEq(t, "MultiError: ", nilMultiErr.Error())

	mult := NewMultiErrorUnchecked([]error{
		errors.New("foo"),
		fs.ErrClosed,
		&exampleErr{"foo", "bar", "baz"},
		errExample,
	})

	assertErrorsIs(t, mult, fs.ErrClosed)

	assertErrorsAs[*exampleErr](t, mult)
	assertErrorsIs(t, mult, errExample)
	assertNotErrorsIs(t, mult, errExampleUnknown)
}

var (
	errExample        = errors.New("example")
	errExampleUnknown = errors.New("unknown")
)

type exampleErr struct {
	Foo string
	Bar string
	Baz string
}

func (e *exampleErr) Error() string {
	if e == nil {
		return "exampleErr: nil"
	}
	return fmt.Sprintf("exampleErr: Foo=%s Bar=%s Baz=%s", e.Foo, e.Bar, e.Baz)
}
