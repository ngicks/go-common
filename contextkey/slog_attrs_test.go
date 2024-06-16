package contextkey

import (
	"bytes"
	"context"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSlogAttrs_Append(t *testing.T) {
	t.Run("basic usage", func(t *testing.T) {
		currentTime := time.Now()

		type testCase struct {
			base   context.Context
			attrs  []slog.Attr
			expect []slog.Attr
		}
		for _, tc := range []testCase{
			{
				base:   context.Background(),
				attrs:  []slog.Attr{slog.String("foo", "bar"), slog.Time("baz", currentTime)},
				expect: []slog.Attr{slog.String("foo", "bar"), slog.Time("baz", currentTime)},
			},
			{
				base:   WithSlogAttrs(context.Background(), []slog.Attr{slog.String("foo", "bar"), slog.Time("baz", currentTime)}),
				attrs:  []slog.Attr{slog.Int("qux", 123), slog.Float64("quux", 1.23)},
				expect: []slog.Attr{slog.String("foo", "bar"), slog.Time("baz", currentTime), slog.Int("qux", 123), slog.Float64("quux", 1.23)},
			},
		} {
			ctx := AppendSlogAttrs(tc.base, tc.attrs...)
			for _, valuer := range []func(ctx context.Context) []slog.Attr{
				func(ctx context.Context) []slog.Attr { v, _ := ValueSlogAttrs(ctx); return v },
				ValueSlogAttrsDefault,
				func(ctx context.Context) []slog.Attr { return ValueSlogAttrsFallback(ctx, nil) },
				func(ctx context.Context) []slog.Attr {
					return ValueSlogAttrsFallbackFn(ctx, func() []slog.Attr { return nil })
				},
			} {
				v := valuer(ctx)
				if !equalSlogAttrs(v, tc.expect) {
					t.Fatalf("")
				}
			}
		}
	})

	// Run this with -race flag.
	t.Run("race", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithSlogAttrs(
			ctx,
			[]slog.Attr{
				slog.String("foo", "bar"),
				slog.String("baz", "qux"),
				slog.Duration("dur", time.Second),
				slog.Float64("num", 1.23),
				slog.Group("gr", slog.String("foo", "bar"), slog.Bool("b", true)),
			},
		)

		var wg sync.WaitGroup
		for range runtime.GOMAXPROCS(0) {
			wg.Add(1)
			go func() {
				ctx := ctx
				defer wg.Done()
				for range 1000 {
					switch rand.N(3) {
					case 0:
						ctx = AppendSlogAttrs(ctx)
					case 1:
						// even number
						ctx = AppendSlogAttrs(
							ctx,
							slog.String("foo", "bar"),
							slog.String("baz", "qux"),
							slog.Duration("dur", time.Second),
							slog.Float64("num", 1.23),
						)
					default:
						// odd number
						ctx = AppendSlogAttrs(
							ctx,
							slog.Any("a", http.DefaultClient),
							slog.Any("b", new(bytes.Buffer)),
							slog.Any("c", new(strings.Builder)),
						)
					}
				}
			}()
		}
		wg.Wait()
	})
}
