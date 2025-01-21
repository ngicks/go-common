package serr

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestUnwrapStack(t *testing.T) {
	base := errors.New("foo")
	first := WithStack(fmt.Errorf("%w", fmt.Errorf("%w", base)))
	err := WithStackOpt(fmt.Errorf("%w", first), &WrapStackOpt{Override: true})

	e := UnwrapStackErr(err)

	if first != errors.Unwrap(e) {
		t.Fatalf("wrong. expected = %#v, actual = %#v", first, e)
	}
}

func TestDeepFrames(t *testing.T) {
	err := foo()

	var funcs []string
	for seq := range DeepFrames(err) {
		funcs = append(funcs, slices.Collect(seq)[0].Function)
	}
	t.Logf("%#v", funcs)
	if !strings.HasSuffix(funcs[0], "bar") {
		t.Fatal("wrong")
	}
	if !strings.HasSuffix(funcs[1], "qux") {
		t.Fatal("wrong")
	}
}

var baseErr = errors.New("base")

func foo() error {
	return bar()
}

func bar() error {
	return WithStackOpt(baz(), &WrapStackOpt{Override: true})
}

func baz() error {
	return qux()
}

func qux() error {
	return WithStackOpt(baseErr, nil)
}
