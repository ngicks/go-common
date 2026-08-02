package atomicsignal

import (
	"os"
	"sync/atomic"
	"testing"
	"time"
)

const testTimeout = 2 * time.Second

type testSignal string

func (s testSignal) Signal() {}

func (s testSignal) String() string {
	return string(s)
}

func receiveSignal(t *testing.T, sink <-chan os.Signal) os.Signal {
	t.Helper()

	select {
	case sig := <-sink:
		return sig
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for signal")
		return nil
	}
}

func runRelay(t *testing.T, relay *Relay) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		relay.Run()
	}()
	t.Cleanup(func() {
		relay.Stop()
		<-done
	})
}

func TestRelayRelaysSignal(t *testing.T) {
	source := make(chan os.Signal, 1)
	sink := make(chan os.Signal, 1)
	relay := newRelay(source, sink, func() {})
	runRelay(t, relay)

	want := testSignal("relay")
	source <- want

	if got := receiveSignal(t, sink); got != want {
		t.Fatalf("received %v, want %v", got, want)
	}
}

func TestRelaySwap(t *testing.T) {
	source := make(chan os.Signal, 1)
	first := make(chan os.Signal, 1)
	second := make(chan os.Signal, 1)
	relay := newRelay(source, first, func() {})
	runRelay(t, relay)

	if previous := relay.Swap(second); previous != chan<- os.Signal(first) {
		t.Fatalf("Swap returned %v, want the first sink", previous)
	}

	want := testSignal("swapped")
	source <- want
	if got := receiveSignal(t, second); got != want {
		t.Fatalf("received %v, want %v", got, want)
	}
	select {
	case got := <-first:
		t.Fatalf("first sink unexpectedly received %v", got)
	default:
	}
}

func TestRelayNilDestinationDropsWithoutStoppingRelay(t *testing.T) {
	source := make(chan os.Signal, 1)
	relay := newRelay(source, nil, func() {})
	t.Cleanup(relay.Stop)

	if running := relay.forward(testSignal("dropped")); !running {
		t.Fatal("relay stopped while dropping a signal")
	}

	sink := make(chan os.Signal, 1)
	if previous := relay.Swap(sink); previous != nil {
		t.Fatalf("Swap returned %v, want nil", previous)
	}

	want := testSignal("delivered")
	if running := relay.forward(want); !running {
		t.Fatal("relay stopped after replacing nil destination")
	}
	if got := receiveSignal(t, sink); got != want {
		t.Fatalf("received %v, want %v", got, want)
	}
}

func TestRelayDoesNotBlockOnUnreadyDestination(t *testing.T) {
	source := make(chan os.Signal, 1)
	unready := make(chan os.Signal)
	relay := newRelay(source, unready, func() {})
	t.Cleanup(relay.Stop)

	if running := relay.forward(testSignal("dropped")); !running {
		t.Fatal("relay stopped while dropping a signal")
	}

	sink := make(chan os.Signal, 1)
	relay.Swap(sink)
	want := testSignal("next")
	if running := relay.forward(want); !running {
		t.Fatal("relay stopped after replacing unready destination")
	}
	if got := receiveSignal(t, sink); got != want {
		t.Fatalf("received %v, want %v", got, want)
	}
}

func TestRelayStopIsIdempotent(t *testing.T) {
	source := make(chan os.Signal, 1)
	var unregisterCalls atomic.Int32
	relay := newRelay(source, nil, func() {
		unregisterCalls.Add(1)
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		relay.Run()
	}()

	relay.Stop()
	relay.Stop()
	<-done

	if got := unregisterCalls.Load(); got != 1 {
		t.Fatalf("unregister called %d times, want 1", got)
	}

	source <- testSignal("after stop")
	sink := make(chan os.Signal, 1)
	relay.Swap(sink)
	select {
	case got := <-sink:
		t.Fatalf("received %v after Stop returned", got)
	default:
	}
}
