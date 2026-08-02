package atomicsignal

import (
	"os"
	"os/signal"
	"sync"
)

// Relay forwards process signals to a replaceable destination.
//
// A Relay must not be copied after first use.
type Relay struct {
	source chan os.Signal

	mu      sync.Mutex
	dest    chan<- os.Signal
	stopped bool

	stopping chan struct{}
	stop     func()
}

// NewRelay registers the provided signals and returns a Relay that forwards
// them to dest.
//
// If no signals are provided, all incoming signals are relayed. NewRelay keeps
// one source registered until [Relay.Stop], so swapping the destination does
// not temporarily restore the process's default signal behavior.
//
// NewRelay does not start a goroutine. Call [Relay.Run] in another goroutine.
// Delivery never blocks; a signal is dropped when dest is nil or not ready.
func NewRelay(dest chan<- os.Signal, signals ...os.Signal) *Relay {
	source := make(chan os.Signal, 32)
	signal.Notify(source, signals...)

	return newRelay(source, dest, func() {
		signal.Stop(source)
	})
}

func newRelay(
	source chan os.Signal,
	dest chan<- os.Signal,
	unregister func(),
) *Relay {
	r := &Relay{
		source:   source,
		dest:     dest,
		stopping: make(chan struct{}),
	}
	r.stop = sync.OnceFunc(func() {
		unregister()
		close(r.stopping)

		r.mu.Lock()
		r.stopped = true
		r.mu.Unlock()
	})
	return r
}

// Run forwards signals until [Relay.Stop] is called.
//
// Run blocks and should normally be called exactly once in another goroutine:
//
//	go relay.Run()
func (r *Relay) Run() {
	for {
		select {
		case <-r.stopping:
			return
		case sig := <-r.source:
			if !r.forward(sig) {
				return
			}
		}
	}
}

func (r *Relay) forward(sig os.Signal) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stopped {
		return false
	}
	select {
	case r.dest <- sig:
	default:
	}
	return true
}

// Swap replaces the destination and returns the previous one.
//
// Swap waits for any in-progress delivery to the previous destination. After
// Swap returns, the Relay no longer uses the previous destination. A nil
// destination discards signals without unregistering the process signal
// handler.
func (r *Relay) Swap(dest chan<- os.Signal) chan<- os.Signal {
	r.mu.Lock()
	defer r.mu.Unlock()

	previous := r.dest
	r.dest = dest
	return previous
}

// Stop unregisters the process signal handler and causes Run to return.
//
// Stop is safe to call more than once. It does not wait for Run to return, but
// after Stop returns no destination is in use.
func (r *Relay) Stop() {
	r.stop()
}
