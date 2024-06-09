package timing

import (
	"context"
	"fmt"
	"sync"
)

type PanicError struct {
	Panicked any
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("process panicked: %v", e.Panicked)
}

// Groups is similar to [errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup)
// but it differentiates itself from errgroup where;
//   - the Group does not limit goroutine number simultaneously runs,
//   - the Group ensures given function is started in a goroutine before return,
//   - allow callers to chose re-panic or convert a panic into an error when fn panics.
//
// Its concern is only about timing and testing.
type Group struct {
	wg       sync.WaitGroup
	mu       sync.Mutex
	repanic  bool
	panicked any
	err      error
	ctx      context.Context
	cancel   context.CancelCauseFunc
}

func NewGroup(ctx context.Context, repanic bool) *Group {
	ctx, cancel := context.WithCancelCause(ctx)
	return &Group{
		ctx:     ctx,
		cancel:  cancel,
		repanic: repanic,
	}
}

func (g *Group) Go(fn func(ctx context.Context) error) {
	// panic in case it already has panicked
	g.mu.Lock()
	if g.repanic && g.panicked != nil {
		p := g.panicked
		g.mu.Unlock()
		panic(p)
	}
	g.mu.Unlock()

	// make sure goroutine created below run before returning from this method.
	switchCh := make(chan struct{})

	g.wg.Add(1)
	go func() {
		<-switchCh
		defer g.wg.Done()
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			g.mu.Lock()
			defer g.mu.Unlock()
			if g.panicked == nil {
				g.panicked = rec
				if g.err == nil {
					g.err = &PanicError{Panicked: rec}
					g.cancel(g.err)
				}
			}
		}()
		err := fn(g.ctx)
		if err != nil {
			g.mu.Lock()
			if g.err == nil {
				g.err = err
				g.cancel(g.err)
			}
			g.mu.Unlock()
		}
	}()

	switchCh <- struct{}{}
}

func (g *Group) Wait() error {
	g.wg.Wait()

	g.mu.Lock()
	defer g.mu.Unlock()
	if g.panicked != nil {
		if g.repanic {
			panic(g)
		} else {
			return &PanicError{Panicked: g.panicked}
		}
	}
	return g.err
}
