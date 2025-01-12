package serr

import (
	"cmp"
	"errors"
	"fmt"
	"testing"
)

var (
	sampleErr1 = errors.New("errors")
	sampleErr2 = &exampleErr{"foo", "bar", "baz"}
)

func TestGather(t *testing.T) {
	for _, errs := range [][]error{
		{nil, nil},
		{},
		nil,
	} {
		assertNilInterface(t, Gather(errs...))
		assertNilInterface(t, GatherChecked(errs...))
		assertBool(t, GatherUnchecked(errs...) != nil, "not NewMultiErrorUnchecked(errs) != nil")
	}

	sampleErrs := []error{sampleErr1, nil, sampleErr2}

	for _, tc := range []struct {
		len int
		fn  func(...error) error
	}{
		{2, Gather},
		{3, GatherChecked},
		{3, GatherUnchecked},
	} {
		err := tc.fn(sampleErrs...)
		assertErrorsIs(t, err, sampleErr1)
		assertErrorsIs(t, err, sampleErr2)
		assertErrorsAs[*exampleErr](t, err)
		assertEq(t, tc.len, len(err.(interface{ Unwrap() []error }).Unwrap()))
	}

	type testCase struct {
		verb     string
		expected string
	}
	for _, tc := range []testCase{
		{verb: "%s", expected: "errors, %!s(<nil>), exampleErr: Foo=foo Bar=bar Baz=baz"},
		{verb: "%v", expected: "errors, <nil>, exampleErr: Foo=foo Bar=bar Baz=baz"},
		{verb: "%+v", expected: "errors, <nil>, exampleErr: Foo=foo Bar=bar Baz=baz"},
		{verb: "%#v", expected: "&errors.errorString{s:\"errors\"}, <nil>, &serr.exampleErr{Foo:\"foo\", Bar:\"bar\", Baz:\"baz\"}"},
		{verb: "%d", expected: "&{%!d(string=errors)}, %!d(<nil>), &{%!d(string=foo) %!d(string=bar) %!d(string=baz)}"},
		{verb: "%T", expected: "*serr.gathered"},
		{verb: "%9.3f", expected: "&{%!f(string=      err)}, %!f(<nil>), &{%!f(string=      foo) %!f(string=      bar) %!f(string=      baz)}"},
	} {
		tc := tc
		t.Run(tc.verb, func(t *testing.T) {
			e := GatherUnchecked(sampleErrs...)
			formatted := fmt.Sprintf(tc.verb, e)
			assertEq(t, tc.expected, formatted)
		})
	}

	nilMultiErr := GatherUnchecked([]error(nil)...)
	assertEq(t, "", nilMultiErr.Error())
}

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

func TestGatherPrefixed(t *testing.T) {
	assertErrorsIs(t, GatherPrefixed(nil), nil)
	assertErrorsIs(t, GatherPrefixed([]PrefixErr{{P: "foo="}, {P: "bar="}}), nil)

	{
		p := GatherPrefixed([]PrefixErr{{P: "foo=", E: sampleErr1}, {P: "bar=", E: sampleErr2}})
		assertErrorsIs(t, p, sampleErr1)
		assertErrorsIs(t, p, sampleErr2)
		assertEq(
			t,
			`foo=&errors.errorString{s:"errors"}, bar=&serr.exampleErr{Foo:"foo", Bar:"bar", Baz:"baz"}`,
			fmt.Sprintf("%#v", p),
		)
	}
	{
		p := GatherPrefixed([]PrefixErr{{P: "foo=", E: sampleErr1}, {P: "bar="}})
		assertErrorsIs(t, p, sampleErr1)
		assertBool(t, !errors.Is(p, sampleErr2), "not !errors.Is(p, sampleErr2)")
		assertEq(
			t,
			`foo=&errors.errorString{s:"errors"}, bar=<nil>`,
			fmt.Sprintf("%#v", p),
		)
	}
}

func TestToPairs(t *testing.T) {
	m := map[string]error{
		"1=": sampleErr1,
		"2=": sampleErr2,
		"3=": nil,
	}
	var pairs []PrefixErr

	pairs = ToPairs(m, nil)

	assertEq(t, pairs[0].P, "1=")
	assertEq(t, pairs[1].P, "2=")
	assertEq(t, pairs[2].P, "3=")

	assertErrorsIs(t, pairs[0].E, sampleErr1)
	assertErrorsIs(t, pairs[1].E, sampleErr2)
	assertErrorsIs(t, pairs[2].E, nil)

	pairs = ToPairs(m, func(i, j string) int { return -cmp.Compare(i, j) })

	assertEq(t, pairs[0].P, "3=")
	assertEq(t, pairs[1].P, "2=")
	assertEq(t, pairs[2].P, "1=")

	assertErrorsIs(t, pairs[0].E, nil)
	assertErrorsIs(t, pairs[1].E, sampleErr2)
	assertErrorsIs(t, pairs[2].E, sampleErr1)
}
