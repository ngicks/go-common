package iopipe

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
)

// Reader pipes a single underlying io.Reader to derived read ends created by
// [Reader.Pipe].
//
// At most one derived pipe is active at a time. Closing a derived pipe pauses
// forwarding without closing or interrupting the underlying reader; bytes
// already pulled from it but not yet delivered are kept for the next derived
// pipe.
type Reader struct {
	src io.Reader

	mu sync.Mutex
	// closed and swapped by new one when a new session arrives
	changed chan struct{}
	active  *rSession
	srcErr  error
	// ended is true when src returned a non-nil error (including io.EOF), or
	// when Run was cancelled.
	ended bool

	pending []byte // owned by Run
}

type rSession struct {
	data      chan []byte
	nread     chan int
	done      chan struct{}
	doneOnce  sync.Once
	cause     error // reason done was closed; written inside doneOnce
	closeErr  chan error
	errOnce   sync.Once
	delivered atomic.Int64
}

func newRSession() *rSession {
	return &rSession{
		data:     make(chan []byte),
		nread:    make(chan int),
		done:     make(chan struct{}),
		closeErr: make(chan error, 1),
	}
}

func (s *rSession) close(cause error) {
	s.doneOnce.Do(func() {
		s.cause = cause
		close(s.done)
	})
}

func (s *rSession) report(err error) {
	s.errOnce.Do(func() {
		s.closeErr <- err
	})
}

// NewReader returns a pipe controller backed by src.
//
// NewReader does not start a goroutine. Call [Reader.Run] in another
// goroutine.
func NewReader(src io.Reader) *Reader {
	return &Reader{
		src:     src,
		changed: make(chan struct{}),
	}
}

// Pipe returns a new derived read end together with a channel reporting how
// it ended.
//
// closeErr receives exactly one value once the derived pipe ends: nil when
// the source was fully drained (it returned io.EOF and every byte reached the
// consumer), or a [*CloseError] otherwise. Cancelling ctx closes the derived
// pipe with the context error.
//
// Pipe returns ErrPipeActive while a previous derived pipe is open, and the
// source's final error (io.EOF included) once the source is exhausted.
func (r *Reader) Pipe(ctx context.Context) (rc io.ReadCloser, closeErr <-chan error, err error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	r.mu.Lock()
	if r.active != nil {
		r.mu.Unlock()
		return nil, nil, ErrPipeActive
	}
	if r.ended {
		err := r.srcErr
		r.mu.Unlock()
		return nil, nil, err
	}
	s := newRSession()
	r.active = s
	r.notifyLocked()
	r.mu.Unlock()

	d := &pipeReader{owner: r, s: s}
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

// Run forwards reads from the underlying reader to derived pipes.
//
// Run blocks until the underlying reader returns an error (io.EOF included)
// and all bytes read before that were delivered, or until ctx is cancelled.
// Cancelling ctx does not interrupt a Read already blocked in the underlying
// reader. Call Run exactly once in another goroutine:
//
//	go reader.Run(ctx)
func (r *Reader) Run(ctx context.Context) {
	buf := make([]byte, 32*1024)
	for {
		s, ok := r.waitSession(ctx)
		if !ok {
			r.stopRun(ctx.Err())
			return
		}

		if len(r.pending) > 0 {
			r.pending = r.deliver(s, r.pending)
			if len(r.pending) > 0 {
				continue
			}
		}

		r.mu.Lock()
		srcErr := r.srcErr
		r.mu.Unlock()
		if srcErr != nil {
			r.finish(s, srcErr)
			return
		}

		n, err := r.src.Read(buf)
		if n > 0 {
			if rest := r.deliver(s, buf[:n]); len(rest) > 0 {
				r.pending = append([]byte(nil), rest...)
				if err != nil {
					r.mu.Lock()
					r.srcErr = err
					r.mu.Unlock()
				}
				continue
			}
		}
		if err != nil {
			r.finish(s, err)
			return
		}
	}
}

// deliver hands p to the session's consumer, returning what was left
// undelivered when the session ended first.
func (r *Reader) deliver(s *rSession, p []byte) []byte {
	for len(p) > 0 {
		select {
		case s.data <- p:
			n := <-s.nread
			p = p[n:]
		case <-s.done:
			return p
		}
	}
	return nil
}

func (r *Reader) finish(s *rSession, err error) {
	r.mu.Lock()
	r.srcErr = err
	r.ended = true
	if r.active == s {
		r.active = nil
	}
	r.mu.Unlock()

	s.close(err)
	if err == io.EOF {
		s.report(nil)
	} else {
		s.report(&CloseError{Delivered: s.delivered.Load(), Cause: err})
	}
}

func (r *Reader) stopRun(err error) {
	r.mu.Lock()
	r.srcErr = err
	r.ended = true
	s := r.active
	r.active = nil
	r.mu.Unlock()

	if s != nil {
		s.close(err)
		s.report(&CloseError{Delivered: s.delivered.Load(), Cause: err})
	}
}

func (r *Reader) detach(s *rSession) {
	r.mu.Lock()
	if r.active == s {
		r.active = nil
	}
	r.mu.Unlock()
}

func (r *Reader) waitSession(ctx context.Context) (*rSession, bool) {
	r.mu.Lock()
	for {
		if ctx.Err() != nil {
			r.mu.Unlock()
			return nil, false
		}
		if r.active != nil {
			s := r.active
			r.mu.Unlock()
			return s, true
		}
		changed := r.changed
		r.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
		}
		r.mu.Lock()
	}
}

func (r *Reader) notifyLocked() {
	close(r.changed)
	r.changed = make(chan struct{})
}

type pipeReader struct {
	owner *Reader
	s     *rSession
	rdMu  sync.Mutex
}

func (d *pipeReader) Read(p []byte) (int, error) {
	d.rdMu.Lock()
	defer d.rdMu.Unlock()

	s := d.s
	select {
	case <-s.done:
		return 0, s.cause
	default:
	}
	select {
	case b := <-s.data:
		n := copy(p, b)
		// Count before unblocking Run so that any subsequent report sees it.
		s.delivered.Add(int64(n))
		s.nread <- n
		return n, nil
	case <-s.done:
		return 0, s.cause
	}
}

func (d *pipeReader) Close() error {
	d.close(io.ErrClosedPipe)
	return nil
}

func (d *pipeReader) close(cause error) {
	s := d.s
	s.close(cause)
	d.owner.detach(s)
	s.report(&CloseError{Delivered: s.delivered.Load(), Cause: s.cause})
}
