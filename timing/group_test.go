package timing

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestGroup(t *testing.T) {
	t.Run("waits until all fn returns", func(t *testing.T) {
		g := NewGroup(context.Background(), false)

		var (
			blocker = make(chan struct{})
		)
		for i := 0; i < 6; i++ {
			g.Go(func(ctx context.Context) error {
				<-blocker
				return nil
			})
		}

		waited := make(chan struct{})
		go func() {
			<-waited
			_ = g.Wait()
			close(waited)
		}()
		waited <- struct{}{}

		for i := 0; i < 5; i++ {
			blocker <- struct{}{}
			time.Sleep(time.Microsecond)
			select {
			case <-waited:
				t.Fatal("Wait unblocked")
			default:
			}
		}

		close(blocker)
		<-waited
	})
	t.Run("first error cancels context and Wait returns first error", func(t *testing.T) {
		g := NewGroup(context.Background(), false)
		fakeErr := errors.New("foobarbaz")
		g.Go(func(ctx context.Context) error {
			return fakeErr
		})
		var (
			ctxErr atomic.Pointer[error]
			cause  atomic.Pointer[error]
		)
		g.Go(func(ctx context.Context) error {
			<-ctx.Done()
			ctxErr_ := ctx.Err()
			cause_ := context.Cause(ctx)
			ctxErr.Store(&ctxErr_)
			cause.Store(&cause_)
			return context.Canceled
		})
		err := g.Wait()
		assert.Assert(t, cmp.ErrorIs(err, fakeErr))
		assert.Assert(t, cmp.ErrorIs(*ctxErr.Load(), context.Canceled))
		assert.Assert(t, cmp.ErrorIs(*cause.Load(), fakeErr))
	})
	t.Run("cancellation propagates", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		g := NewGroup(ctx, false)

		var (
			blocker = make(chan struct{})
			ctxErr  atomic.Pointer[error]
			cause   atomic.Pointer[error]
			waitErr atomic.Pointer[error]
		)
		g.Go(func(ctx context.Context) error {
			<-blocker
			ctxErr_ := ctx.Err()
			cause_ := context.Cause(ctx)
			ctxErr.Store(&ctxErr_)
			cause.Store(&cause_)
			return fmt.Errorf("foobarbaz")
		})

		waited := make(chan struct{})
		go func() {
			<-waited
			waitErr_ := g.Wait()
			waitErr.Store(&waitErr_)
			close(waited)
		}()
		waited <- struct{}{}

		for i := 0; i < 5; i++ {
			time.Sleep(time.Microsecond)
			select {
			case <-waited:
				t.Fatal("Wait unblocked")
			default:
			}
		}

		cancel(fmt.Errorf("bazbazbaz"))
		close(blocker)
		<-waited

		assert.Assert(t, cmp.ErrorIs(*ctxErr.Load(), context.Canceled))
		assert.ErrorContains(t, *cause.Load(), "bazbazbaz")
		assert.ErrorContains(t, *waitErr.Load(), "foobarbaz")
	})

	t.Run("repanic", func(t *testing.T) {
		g := NewGroup(context.Background(), false)
		g.Go(func(ctx context.Context) error {
			panic(fmt.Errorf("foobarbaz"))
		})
		err := g.Wait().(*PanicError)
		assert.ErrorContains(t, err.Panicked.(error), "foobarbaz")

		g = NewGroup(context.Background(), true)
		g.Go(func(ctx context.Context) error {
			panic(fmt.Errorf("foobarbaz"))
		}) // this does not kill the process
		assert.Assert(t, cmp.Panics(func() {
			g.Go(func(ctx context.Context) error {
				return nil
			})
		}))
		assert.Assert(t, cmp.Panics(func() {
			_ = g.Wait()
		}))
	})
}
