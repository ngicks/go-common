package atomicsignal

import (
	"os"
	"sync"
)

// Handler dispatches signals received by its destination channel.
//
// A Handler must not be copied after first use.
type Handler struct {
	dest           chan os.Signal
	defaultHandler func(sig os.Signal)
	handlers       map[os.Signal]func(sig os.Signal)

	stopping chan struct{}
	stop     func()
}

// NewHandler returns a Handler with a destination channel of the given buffer
// size.
//
// A signal with a matching entry in handlers is passed to that function.
// Otherwise it is passed to defaultHandler. A nil selected function discards
// the signal.
//
// NewHandler does not start a goroutine. Call [Handler.Run] in another
// goroutine.
func NewHandler(
	buffer int,
	defaultHandler func(sig os.Signal),
	handlers map[os.Signal]func(sig os.Signal),
) *Handler {
	copiedHandlers := make(map[os.Signal]func(sig os.Signal), len(handlers))
	for sig, handler := range handlers {
		copiedHandlers[sig] = handler
	}

	h := &Handler{
		dest:           make(chan os.Signal, buffer),
		defaultHandler: defaultHandler,
		handlers:       copiedHandlers,
		stopping:       make(chan struct{}),
	}
	h.stop = sync.OnceFunc(func() {
		close(h.stopping)
	})
	return h
}

// Dest returns the channel to which signals should be sent.
func (h *Handler) Dest() chan<- os.Signal {
	return h.dest
}

// Run dispatches signals until [Handler.Stop] is called or the destination is
// closed.
//
// Run blocks and should normally be called exactly once in another goroutine:
//
//	go handler.Run()
func (h *Handler) Run() {
	for {
		select {
		case <-h.stopping:
			return
		case sig, ok := <-h.dest:
			if !ok {
				return
			}
			h.handle(sig)
		}
	}
}

func (h *Handler) handle(sig os.Signal) {
	handler, ok := h.handlers[sig]
	if !ok {
		handler = h.defaultHandler
	}
	if handler != nil {
		handler(sig)
	}
}

// Stop causes Run to return. It is safe to call Stop more than once.
func (h *Handler) Stop() {
	h.stop()
}
