package testhelper

import (
	"errors"
	"testing"
)

func AssertErrorsIs(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("errors.Is(err, target) returned false, err = %#v, target = %#v", err, target)
	}
}

func AssertNotErrorsIs(t *testing.T, err, target error) {
	t.Helper()
	if errors.Is(err, target) {
		t.Fatalf("errors.Is(err, target) returned true, err = %#v, target = %#v", err, target)
	}
}

func AssertErrorsAs[T any](t *testing.T, err error) {
	t.Helper()
	var e T
	if !errors.As(err, &e) {
		t.Fatalf("errors.As(err, target) returned false, expected to be type %T, but is %#v", e, err)
	}
}

func AssertNilInterface(t *testing.T, v any) {
	t.Helper()
	if v != nil {
		t.Fatalf("not nil: v = %#v,\nexpected to be nil", v)
	}
}

func AssertTrue(t *testing.T, b bool, format string, mgsArgs ...any) {
	t.Helper()
	if !b {
		t.Fatalf("not true: "+format, mgsArgs...)
	}
}

func AssertFalse(t *testing.T, b bool, format string, mgsArgs ...any) {
	t.Helper()
	if b {
		t.Fatalf("not false: "+format, mgsArgs...)
	}
}

func AssertEq[T comparable](t *testing.T, x, y T) {
	t.Helper()
	if x != y {
		t.Fatalf("not equal: left =\n%v,\n\nright =\n%v", x, y)
	}
}
