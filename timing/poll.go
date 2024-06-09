package timing

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/jonboulle/clockwork"
)

type pollParam struct {
	ctx   context.Context
	clock clockwork.Clock
}

func newPollParam() *pollParam {
	return &pollParam{
		ctx:   context.Background(),
		clock: clockwork.NewRealClock(),
	}
}

type pollOption func(*pollParam)

func SetPollContext(ctx context.Context) pollOption {
	return func(pp *pollParam) {
		pp.ctx = ctx
	}
}

// PollUntil calls predicate multiple times periodically at interval until it returns true,
// or timeout duration since the invocation of this function is passed.
func PollUntil(predicate func(ctx context.Context) bool, interval time.Duration, timeout time.Duration, options ...pollOption) (ok bool) {
	param := newPollParam()

	for _, opt := range options {
		opt(param)
	}

	ctx := param.ctx

	predCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	doneCh := make(chan struct{})

	defer func() {
		<-doneCh
	}()

	var called atomic.Bool
	done := func() {
		if called.CompareAndSwap(false, true) {
			cancel()
			close(doneCh)
		}
	}

	wait := make(chan struct{})
	defer func() {
		<-wait
	}()

	go func() {
		defer func() { close(wait) }()
		t := param.clock.NewTimer(time.Hour)
		_ = t.Stop()
		defer t.Stop()
		for {
			select {
			case <-doneCh:
				return
			default:
			}
			if predicate(predCtx) {
				break
			}
			t.Reset(interval) // t is known emitted or stopped.
			select {
			case <-t.Chan():
			case <-doneCh:
				return
			}
		}
		done()
	}()

	t := param.clock.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-ctx.Done():
		done()
		return false
	case <-t.Chan():
		done()
		return false
	default:
		select {
		case <-ctx.Done():
			done()
			return false
		case <-t.Chan():
			done()
			return false
		case <-doneCh:
			return true
		}
	}
}
