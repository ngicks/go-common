// Package iopipe provides io.Pipe-like pipes in front of an io.Reader or
// io.Writer that make byte consumption observable.
//
// A plain io.Pipe cannot tell the orchestrating side whether piped bytes were
// consumed once an error closes the pipe. [Reader] and [Writer] keep the piped
// ends as plain io.ReadCloser / io.WriteCloser and instead report the outcome
// on a per-pipe channel returned by Pipe: it receives exactly one value — nil
// when everything sent through the pipe was consumed, or a [*CloseError]
// carrying the delivered byte count and the cause.
//
// Closing a derived pipe pauses forwarding without closing the underlying
// reader or writer, and an operation already blocked in the underlying reader
// or writer is never interrupted. At most one derived pipe is active at a
// time; Pipe can be called again after the previous one is closed. On the
// reader side, bytes already pulled from the source but not yet delivered
// carry over to the next derived pipe.
package iopipe
