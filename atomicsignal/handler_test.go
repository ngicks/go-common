package atomicsignal

import (
	"os"
	"testing"
	"time"
)

func runHandler(t *testing.T, handler *Handler) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		handler.Run()
	}()
	t.Cleanup(func() {
		handler.Stop()
		<-done
	})
}

func TestHandlerDispatchesSpecificAndDefaultHandlers(t *testing.T) {
	specificSignal := testSignal("specific")
	defaultSignal := testSignal("default")
	handled := make(chan string, 2)
	handler := NewHandler(
		2,
		func(sig os.Signal) {
			handled <- "default:" + sig.String()
		},
		map[os.Signal]func(os.Signal){
			specificSignal: func(sig os.Signal) {
				handled <- "specific:" + sig.String()
			},
		},
	)
	runHandler(t, handler)

	handler.Dest() <- specificSignal
	handler.Dest() <- defaultSignal

	if got := <-handled; got != "specific:specific" {
		t.Fatalf("handled %q, want specific handler", got)
	}
	if got := <-handled; got != "default:default" {
		t.Fatalf("handled %q, want default handler", got)
	}
}

func TestHandlerCopiesHandlerMap(t *testing.T) {
	sig := testSignal("copied")
	called := make(chan struct{}, 1)
	handlers := map[os.Signal]func(os.Signal){
		sig: func(os.Signal) {
			called <- struct{}{}
		},
	}
	handler := NewHandler(1, nil, handlers)
	delete(handlers, sig)
	runHandler(t, handler)

	handler.Dest() <- sig
	select {
	case <-called:
	case <-time.After(testTimeout):
		t.Fatal("copied handler was not called")
	}
}
