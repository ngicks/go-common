package iopipe

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
)

// Writer pipes derived write ends created by [Writer.Pipe] to a single
// underlying io.Writer.
//
// At most one derived pipe is active at a time. A derived Write hands bytes
// through synchronously: it returns only after the underlying writer accepted
// them, so its count is exact. Closing a derived pipe does not close the
// underlying writer.
type Writer struct {
	dst io.Writer

	mu      sync.Mutex
	changed chan struct{}
	active  *wSession
	dstErr  error
	// ended is true when dst returned a non-nil error, or when Run was
	// cancelled.
	ended bool
}

type writeRes struct {
	n   int
	err error
}

type wSession struct {
	data     chan []byte
	res      chan writeRes
	done     chan struct{}
	doneOnce sync.Once
	cause    error // reason done was closed; written inside doneOnce
	closeErr chan error
	errOnce  sync.Once
	flushed  atomic.Int64
}

func newWSession() *wSession {
	return &wSession{
		data:     make(chan []byte),
		res:      make(chan writeRes),
		done:     make(chan struct{}),
		closeErr: make(chan error, 1),
	}
}

func (s *wSession) close(cause error) {
	s.doneOnce.Do(func() {
		s.cause = cause
		close(s.done)
	})
}

func (s *wSession) report(err error) {
	s.errOnce.Do(func() {
		s.closeErr <- err
	})
}

func (s *wSession) closedErr() error {
	if s.cause == nil {
		return io.ErrClosedPipe
	}
	return s.cause
}

// NewWriter returns a pipe controller backed by dst.
//
// NewWriter does not start a goroutine. Call [Writer.Run] in another
// goroutine.
func NewWriter(dst io.Writer) *Writer {
	return &Writer{
		dst:     dst,
		changed: make(chan struct{}),
	}
}

// Pipe returns a new derived write end together with a channel reporting how
// it ended.
//
// closeErr receives exactly one value once the derived pipe ends: nil when
// the derived pipe was closed cleanly (every byte its Write accepted reached
// the underlying writer), or a [*CloseError] otherwise. Cancelling ctx closes
// the derived pipe with the context error; a Write already in flight is not
// interrupted.
//
// Pipe returns ErrPipeActive while a previous derived pipe is open, and the
// underlying writer's error once it has failed.
func (w *Writer) Pipe(ctx context.Context) (wc io.WriteCloser, closeErr <-chan error, err error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	w.mu.Lock()
	if w.active != nil {
		w.mu.Unlock()
		return nil, nil, ErrPipeActive
	}
	if w.ended {
		err := w.dstErr
		w.mu.Unlock()
		return nil, nil, err
	}
	s := newWSession()
	w.active = s
	w.notifyLocked()
	w.mu.Unlock()

	d := &pipeWriter{owner: w, s: s}
	if done := ctx.Done(); done != nil {
		go func() {
			select {
			case <-done:
				d.close(ctx.Err())
			case <-s.done:
			}
		}()
	}
	return d, s.closeErr, nil
}

// Run forwards writes from derived pipes to the underlying writer.
//
// Run blocks until the underlying writer returns an error or ctx is
// cancelled. Cancelling ctx does not interrupt a Write already blocked in the
// underlying writer. Call Run exactly once in another goroutine:
//
//	go writer.Run(ctx)
func (w *Writer) Run(ctx context.Context) {
	for {
		s, ok := w.waitSession(ctx)
		if !ok {
			w.stopRun(ctx.Err())
			return
		}
		select {
		case p := <-s.data:
			n, err := writeAll(w.dst, p)
			s.flushed.Add(int64(n))
			s.res <- writeRes{n: n, err: err}
			if err != nil {
				w.finish(s, err)
				return
			}
		case <-s.done:
			w.detach(s)
		}
	}
}

func (w *Writer) finish(s *wSession, err error) {
	w.mu.Lock()
	w.dstErr = err
	w.ended = true
	if w.active == s {
		w.active = nil
	}
	w.mu.Unlock()

	s.close(err)
	s.report(&CloseError{Delivered: s.flushed.Load(), Cause: err})
}

func (w *Writer) stopRun(err error) {
	w.mu.Lock()
	w.dstErr = err
	w.ended = true
	s := w.active
	w.active = nil
	w.mu.Unlock()

	if s != nil {
		s.close(err)
		s.report(&CloseError{Delivered: s.flushed.Load(), Cause: err})
	}
}

func (w *Writer) detach(s *wSession) {
	w.mu.Lock()
	if w.active == s {
		w.active = nil
	}
	w.mu.Unlock()
}

func (w *Writer) waitSession(ctx context.Context) (*wSession, bool) {
	w.mu.Lock()
	for {
		if ctx.Err() != nil {
			w.mu.Unlock()
			return nil, false
		}
		if w.active != nil {
			s := w.active
			w.mu.Unlock()
			return s, true
		}
		changed := w.changed
		w.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
		}
		w.mu.Lock()
	}
}

func (w *Writer) notifyLocked() {
	close(w.changed)
	w.changed = make(chan struct{})
}

type pipeWriter struct {
	owner *Writer
	s     *wSession
	wrMu  sync.Mutex
}

func (d *pipeWriter) Write(p []byte) (int, error) {
	d.wrMu.Lock()
	defer d.wrMu.Unlock()

	s := d.s
	select {
	case <-s.done:
		return 0, s.closedErr()
	default:
	}
	var n int
	for once := true; once || len(p) > 0; once = false {
		select {
		case s.data <- p:
			res := <-s.res
			n += res.n
			p = p[res.n:]
			if res.err != nil {
				return n, res.err
			}
		case <-s.done:
			return n, s.closedErr()
		}
	}
	return n, nil
}

func (d *pipeWriter) Close() error {
	d.close(nil)
	return nil
}

// close takes wrMu so that an in-flight Write settles first; a clean-close
// report therefore never precedes a late flush failure.
func (d *pipeWriter) close(cause error) {
	d.wrMu.Lock()
	defer d.wrMu.Unlock()

	s := d.s
	s.close(cause)
	d.owner.detach(s)
	if s.cause == nil {
		s.report(nil)
	} else {
		s.report(&CloseError{Delivered: s.flushed.Load(), Cause: s.cause})
	}
}

func writeAll(dst io.Writer, p []byte) (int, error) {
	var n int
	for n < len(p) {
		m, err := dst.Write(p[n:])
		n += m
		if err != nil {
			return n, err
		}
		if m == 0 {
			return n, io.ErrShortWrite
		}
	}
	return n, nil
}
