package iopipe

import (
	"errors"
	"fmt"
)

// ErrPipeActive is returned by Pipe while a previously derived pipe is still
// open.
var ErrPipeActive = errors.New("iopipe: derived pipe is still active")

// CloseError reports how a derived pipe ended when full consumption was not
// confirmed.
//
// Delivered counts the bytes that made it through the derived pipe: bytes
// returned by the consumer's Read for [Reader], bytes accepted by the
// underlying writer for [Writer]. Cause is the reason the pipe ended:
// io.ErrClosedPipe when the derived pipe was closed by its user, the context
// error when it was cancelled, or the underlying reader's / writer's error.
type CloseError struct {
	Delivered int64
	Cause     error
}

func (e *CloseError) Error() string {
	return fmt.Sprintf("iopipe: pipe closed after %d bytes: %v", e.Delivered, e.Cause)
}

func (e *CloseError) Unwrap() error {
	return e.Cause
}
