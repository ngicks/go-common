// Package atomicsignal relays process signals to a replaceable destination and
// dispatches signals to handlers.
//
// A Relay keeps one channel registered with [os/signal] for its lifetime.
// Replacing its destination therefore does not create a window in which the
// process signal disposition falls back to its default.
package atomicsignal
