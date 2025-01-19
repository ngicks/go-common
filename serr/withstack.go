package serr

import (
	"errors"
	"fmt"
	"io"
	"iter"
	"runtime"
	"strings"
)

type withStack struct {
	err error
	pc  []uintptr
}

const (
	defaultStackSizeInitial = 64
	defaultStackSizeMax     = 1 << 11
)

func wrapStack(err error, override bool, depth int) error {
	if !override {
		var ws *withStack
		if errors.As(err, &ws) {
			// already wrapped
			return err
		}
	}
	initialSize := defaultStackSizeInitial
	maxSize := defaultStackSizeMax
	if depth > 0 {
		initialSize = min(defaultStackSizeInitial, depth+1)
		maxSize = depth + 1
	}
	pc := make([]uintptr, initialSize)
	// skip runtime.Callers, WithStack|WithStackOverride, wrapStack
	n := runtime.Callers(3, pc)
	if maxSize != initialSize {
		for n == len(pc) {
			// grow. let append decide size.
			pc = append(pc, 0)
			pc = pc[:cap(pc)]
			n = runtime.Callers(3, pc)
			if n >= maxSize {
				break
			}
		}
	}

	return &withStack{
		err: err,
		pc:  pc[:min(n, maxSize)],
	}
}

// WithStack adds stacktrace information to err using [runtime.Callers].
//
// WithStack wraps err only when it has not yet been wrapped.
// The depth of the stack trace is limited to 64.
// To control these behavior, use [WithStackOverride].
func WithStack(err error) error {
	return wrapStack(err, false, defaultStackSizeInitial)
}

// WithStackOverride is like [WithStack] but allows to control override and/or depth of stacktrace.
//
// WithStackOverride returns err without doing anything if override is false.
// depth controls max number of stack frames embedded into the returned error.
// If depth is less than or equals to 0, then it is limited to 2048.
func WithStackOverride(err error, override bool, depth int) error {
	return wrapStack(err, override, depth)
}

func (e *withStack) format(w io.Writer, format string) {
	_, _ = fmt.Fprintf(w, format, e.err)
}

func (e *withStack) Error() string {
	var s strings.Builder
	e.format(&s, "%s")
	return s.String()
}

func (e *withStack) Format(state fmt.State, verb rune) {
	e.format(state, fmt.FormatString(state, verb))
}

func (e *withStack) Unwrap() error {
	return e.err
}

// Pc retrieves slice of pc from err
// The slice is nil if err has not been wrapped by [WithStack] or [WithStackOverride].
func Pc(err error) []uintptr {
	var ws *withStack
	if !errors.As(err, &ws) {
		return nil
	}
	return ws.pc
}

// Frames returns an iterator over [runtime.Frame] using pc embedded to err.
// The iterator yields nothing if err has not been wrapped by [WithStack] or [WithStackOverride].
func Frames(err error) iter.Seq[runtime.Frame] {
	return func(yield func(runtime.Frame) bool) {
		pc := Pc(err)
		if len(pc) == 0 {
			return
		}
		frames := runtime.CallersFrames(pc)
		for {
			f, ok := frames.Next()
			if !ok {
				return
			}
			if !yield(f) {
				return
			}
		}
	}
}

// PrintStack writes each stack frame information retrieved from err into w.
func PrintStack(w io.Writer, err error) error {
	for f := range Frames(err) {
		_, err := fmt.Fprintf(w, "%s(%s:%d)\n", f.Function, f.File, f.Line)
		if err != nil {
			return err
		}
	}
	return nil
}
