package serr_test

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"os"
	"path"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/ngicks/go-common/serr"
)

func mapIter[V1, V2 any](mapper func(V1) V2, seq iter.Seq[V1]) iter.Seq[V2] {
	return func(yield func(V2) bool) {
		for v := range seq {
			if !yield(mapper(v)) {
				return
			}
		}
	}
}

func frameToStr(f runtime.Frame) string {
	// take base for stability
	fn := f.Function
	fn = strings.TrimPrefix(fn, "github.com/ngicks/go-common/serr_test.")
	fn = strings.TrimPrefix(fn, "command-line-arguments_test.")
	if fn == "main.main" {
		f.Line = 63 // hack: specifying test target as file v.s. package changes this line number.
	}
	return fmt.Sprintf("%s(%s:%d)", fn, path.Base(f.File), f.Line)
}

func f1() {
	f2()
}

func f2() {
	f3()
}

func f3() {
	var (
		panicVal  any
		panicOnce sync.Once
	)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			panicOnce.Do(func() {
				panicVal = serr.WithStackOpt(fmt.Errorf("%v", rec), &serr.WrapStackOpt{Override: true})
			})
		}()
		f4()
	}()
	<-ctx.Done()
	if panicVal != nil {
		panic(panicVal)
	}
}

func f4() {
	f5()
}

func f5() {
	f6()
}

func f6() {
	s := make([]int, 2)
	_ = s[4]
}

// Example_multiple_stack demonstrates use case of [serr.WithStack] and [serr.DeepFrames]
func Example_multiple_stack() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == "time" {
				a.Value = slog.StringValue("2025-01-22T11:37:12.566633474Z")
			}
			return a
		},
	}))
	defer func() {
		rec := recover()
		if rec == nil {
			return
		}
		err, ok := rec.(error)
		if !ok {
			panic(rec)
		}
		err = serr.WithStackOpt(err, &serr.WrapStackOpt{Override: true})
		logger.Error(
			"panicked",
			slog.Any(
				"stack_trace",
				slices.Collect(
					mapIter(
						func(seq iter.Seq[runtime.Frame]) []string {
							return slices.Collect(mapIter(frameToStr, seq))
						},
						serr.DeepFrames(err),
					),
				),
			),
		)
	}()
	f1()
	// Output:
	// {"time":"2025-01-22T11:37:12.566633474Z","level":"ERROR","msg":"panicked","stack_trace":[["Example_multiple_stack.func2(example_multiple_stack_test.go:104)","runtime.gopanic(panic.go:785)","f3(example_multiple_stack_test.go:68)","f2(example_multiple_stack_test.go:44)","f1(example_multiple_stack_test.go:40)","Example_multiple_stack(example_multiple_stack_test.go:120)","testing.runExample(run_example.go:63)","testing.runExamples(example.go:40)","testing.(*M).Run(testing.go:2036)","main.main(_testmain.go:63)","runtime.main(proc.go:272)"],["f3.func1.1.1(example_multiple_stack_test.go:61)","sync.(*Once).doSlow(once.go:76)","sync.(*Once).Do(once.go:67)","f3.func1.1(example_multiple_stack_test.go:60)","runtime.gopanic(panic.go:785)","runtime.goPanicIndex(panic.go:115)","f6(example_multiple_stack_test.go:82)","f5(example_multiple_stack_test.go:77)","f4(example_multiple_stack_test.go:73)","f3.func1(example_multiple_stack_test.go:64)"]]}
}
