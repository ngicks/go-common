package serr

import (
	"errors"
	"fmt"
	"testing"
)

func TestPrefix(t *testing.T) {
	assertErrorsIs(t, Prefix("foo: ", nil), nil)
	assertErrorsAs[*prefixed](t, PrefixUnchecked("foo: ", nil))

	err := errors.New("sample")
	prefixed := Prefix("foo: ", err)
	assertErrorsIs(t, Prefix("foo: ", err), err)

	assertEq(t, "foo: sample", prefixed.Error())
	assertEq(t, "foo: &errors.errorString{s:\"sample\"}", fmt.Sprintf("%#v", prefixed))
}
