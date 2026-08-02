package iopipe

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

func TestReader_full_consumption_reports_nil(t *testing.T) {
	r := NewReader(strings.NewReader("hello world"))
	go r.Run(t.Context())

	rc, closeErr, err := r.Pipe(context.Background())
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(b) != "hello world" {
		t.Fatalf("read %q", b)
	}
	if err := <-closeErr; err != nil {
		t.Fatalf("closeErr = %v, want nil", err)
	}

	if _, _, err := r.Pipe(context.Background()); err != io.EOF {
		t.Fatalf("Pipe after exhaustion = %v, want io.EOF", err)
	}
}

func TestReader_early_close_reports_delivered_and_resumes(t *testing.T) {
	srcR, srcW := io.Pipe()
	r := NewReader(srcR)
	go r.Run(t.Context())
	go func() { _, _ = srcW.Write([]byte("hello world")) }()

	rc1, closeErr1, err := r.Pipe(context.Background())
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	buf := make([]byte, 5)
	if _, err := io.ReadFull(rc1, buf); err != nil {
		t.Fatalf("ReadFull: %v", err)
	}
	if string(buf) != "hello" {
		t.Fatalf("read %q", buf)
	}
	_ = rc1.Close()

	var ce *CloseError
	if err := <-closeErr1; !errors.As(err, &ce) {
		t.Fatalf("closeErr = %v, want *CloseError", err)
	}
	if ce.Delivered != 5 || !errors.Is(ce, io.ErrClosedPipe) {
		t.Fatalf("CloseError = %+v", ce)
	}
	if _, err := rc1.Read(buf); err != io.ErrClosedPipe {
		t.Fatalf("Read after Close = %v, want io.ErrClosedPipe", err)
	}

	rc2, closeErr2, err := r.Pipe(context.Background())
	if err != nil {
		t.Fatalf("re-Pipe: %v", err)
	}
	_ = srcW.Close()
	b, err := io.ReadAll(rc2)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(b) != " world" {
		t.Fatalf("resumed read %q", b)
	}
	if err := <-closeErr2; err != nil {
		t.Fatalf("closeErr2 = %v, want nil", err)
	}
}

func TestReader_src_error_propagates(t *testing.T) {
	srcErr := errors.New("src broke")
	r := NewReader(io.MultiReader(strings.NewReader("hello"), iotest.ErrReader(srcErr)))
	go r.Run(t.Context())

	rc, closeErr, err := r.Pipe(context.Background())
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	buf := make([]byte, 5)
	if _, err := io.ReadFull(rc, buf); err != nil {
		t.Fatalf("ReadFull: %v", err)
	}
	if _, err := rc.Read(buf); err != srcErr {
		t.Fatalf("Read = %v, want %v", err, srcErr)
	}

	var ce *CloseError
	if err := <-closeErr; !errors.As(err, &ce) {
		t.Fatalf("closeErr = %v, want *CloseError", err)
	}
	if ce.Delivered != 5 || ce.Cause != srcErr {
		t.Fatalf("CloseError = %+v", ce)
	}
	if _, _, err := r.Pipe(context.Background()); err != srcErr {
		t.Fatalf("Pipe after src error = %v, want %v", err, srcErr)
	}
}

func TestReader_cancel_unblocks_read(t *testing.T) {
	srcR, _ := io.Pipe() // never written; keeps Run blocked in src.Read
	r := NewReader(srcR)
	go r.Run(t.Context())

	ctx, cancel := context.WithCancel(context.Background())
	rc, closeErr, err := r.Pipe(ctx)
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	readErr := make(chan error, 1)
	go func() {
		_, err := rc.Read(make([]byte, 1))
		readErr <- err
	}()
	cancel()
	if err := <-readErr; err != context.Canceled {
		t.Fatalf("Read = %v, want context.Canceled", err)
	}

	var ce *CloseError
	if err := <-closeErr; !errors.As(err, &ce) {
		t.Fatalf("closeErr = %v, want *CloseError", err)
	}
	if ce.Delivered != 0 || ce.Cause != context.Canceled {
		t.Fatalf("CloseError = %+v", ce)
	}
}

func TestReader_second_pipe_rejected_until_close(t *testing.T) {
	r := NewReader(strings.NewReader("data"))
	go r.Run(t.Context())

	rc, _, err := r.Pipe(context.Background())
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	if _, _, err := r.Pipe(context.Background()); err != ErrPipeActive {
		t.Fatalf("second Pipe = %v, want ErrPipeActive", err)
	}
	_ = rc.Close()
	if _, _, err := r.Pipe(context.Background()); err != nil {
		t.Fatalf("Pipe after Close = %v", err)
	}
}

func TestRun_cancel_stops_pump(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r := NewReader(strings.NewReader("data"))
	rDone := make(chan struct{})
	go func() {
		r.Run(ctx)
		close(rDone)
	}()
	w := NewWriter(io.Discard)
	wDone := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(wDone)
	}()

	cancel()
	<-rDone
	<-wDone
	if _, _, err := r.Pipe(context.Background()); err != context.Canceled {
		t.Fatalf("Reader.Pipe after Run cancel = %v, want context.Canceled", err)
	}
	if _, _, err := w.Pipe(context.Background()); err != context.Canceled {
		t.Fatalf("Writer.Pipe after Run cancel = %v, want context.Canceled", err)
	}
}

func TestWriter_clean_close_reports_nil_and_reuses(t *testing.T) {
	var dst bytes.Buffer
	w := NewWriter(&dst)
	go w.Run(t.Context())

	wc1, closeErr1, err := w.Pipe(context.Background())
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	n, err := wc1.Write([]byte("hello world"))
	if n != 11 || err != nil {
		t.Fatalf("Write = %d, %v", n, err)
	}
	_ = wc1.Close()
	if err := <-closeErr1; err != nil {
		t.Fatalf("closeErr1 = %v, want nil", err)
	}
	if _, err := wc1.Write([]byte("x")); err != io.ErrClosedPipe {
		t.Fatalf("Write after Close = %v, want io.ErrClosedPipe", err)
	}

	wc2, closeErr2, err := w.Pipe(context.Background())
	if err != nil {
		t.Fatalf("re-Pipe: %v", err)
	}
	if _, err := wc2.Write([]byte("!")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	_ = wc2.Close()
	if err := <-closeErr2; err != nil {
		t.Fatalf("closeErr2 = %v, want nil", err)
	}
	if dst.String() != "hello world!" {
		t.Fatalf("dst = %q", dst.String())
	}
}

// failWriter accepts `accept` bytes then fails every Write with err.
type failWriter struct {
	accept int
	err    error
}

func (f *failWriter) Write(p []byte) (int, error) {
	if f.accept <= 0 {
		return 0, f.err
	}
	n := min(len(p), f.accept)
	f.accept -= n
	if n < len(p) {
		return n, f.err
	}
	return n, nil
}

func TestWriter_dst_error_reports_partial_write(t *testing.T) {
	dstErr := errors.New("dst broke")
	w := NewWriter(&failWriter{accept: 5, err: dstErr})
	go w.Run(t.Context())

	wc, closeErr, err := w.Pipe(context.Background())
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	n, err := wc.Write([]byte("hello world"))
	if n != 5 || err != dstErr {
		t.Fatalf("Write = %d, %v; want 5, %v", n, err, dstErr)
	}

	var ce *CloseError
	if err := <-closeErr; !errors.As(err, &ce) {
		t.Fatalf("closeErr = %v, want *CloseError", err)
	}
	if ce.Delivered != 5 || ce.Cause != dstErr {
		t.Fatalf("CloseError = %+v", ce)
	}
	if _, err := wc.Write([]byte("x")); err != dstErr {
		t.Fatalf("Write after dst error = %v, want %v", err, dstErr)
	}
	if _, _, err := w.Pipe(context.Background()); err != dstErr {
		t.Fatalf("Pipe after dst error = %v, want %v", err, dstErr)
	}
}

func TestWriter_cancel_reports_and_fails_writes(t *testing.T) {
	var dst bytes.Buffer
	w := NewWriter(&dst)
	go w.Run(t.Context())

	ctx, cancel := context.WithCancel(context.Background())
	wc, closeErr, err := w.Pipe(ctx)
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	cancel()

	var ce *CloseError
	if err := <-closeErr; !errors.As(err, &ce) {
		t.Fatalf("closeErr = %v, want *CloseError", err)
	}
	if ce.Delivered != 0 || ce.Cause != context.Canceled {
		t.Fatalf("CloseError = %+v", ce)
	}
	if _, err := wc.Write([]byte("x")); err != context.Canceled {
		t.Fatalf("Write after cancel = %v, want context.Canceled", err)
	}
}

func TestWriter_second_pipe_rejected_until_close(t *testing.T) {
	var dst bytes.Buffer
	w := NewWriter(&dst)
	go w.Run(t.Context())

	wc, _, err := w.Pipe(context.Background())
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	if _, _, err := w.Pipe(context.Background()); err != ErrPipeActive {
		t.Fatalf("second Pipe = %v, want ErrPipeActive", err)
	}
	_ = wc.Close()
	if _, _, err := w.Pipe(context.Background()); err != nil {
		t.Fatalf("Pipe after Close = %v", err)
	}
}
