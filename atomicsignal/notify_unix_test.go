//go:build unix

package atomicsignal

import (
	"os"
	"syscall"
	"testing"
)

func TestRelayRelaysRuntimeSignal(t *testing.T) {
	sink := make(chan os.Signal, 1)
	relay := NewRelay(sink, syscall.SIGUSR1)
	runRelay(t, relay)

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGUSR1); err != nil {
		t.Fatalf("failed to send signal: %v", err)
	}

	if got := receiveSignal(t, sink); got != syscall.SIGUSR1 {
		t.Fatalf("received %v, want %v", got, syscall.SIGUSR1)
	}
}
