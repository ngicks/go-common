# atomicsignal

`atomicsignal` keeps a single `os/signal` source registered while allowing its
destination to be replaced. This avoids the default-signal window created by
stopping one runtime signal channel before registering another.

```go
handler := atomicsignal.NewHandler(
	1,
	func(sig os.Signal) {
		log.Printf("unhandled signal: %v", sig)
	},
	map[os.Signal]func(os.Signal){
		syscall.SIGTERM: func(sig os.Signal) {
			cancel()
		},
	},
)
go handler.Run()
defer handler.Stop()

relay := atomicsignal.NewRelay(
	handler.Dest(),
	os.Interrupt,
	syscall.SIGTERM,
)
go relay.Run()
defer relay.Stop()

childSignals := make(chan os.Signal, 1)
previous := relay.Swap(childSignals)

// Restore the earlier destination. Once Swap returns, childSignals is no
// longer used and can safely be closed.
relay.Swap(previous)
```

Destinations should be buffered. Delivery is non-blocking, matching the
best-effort nature of `os/signal`; a signal is dropped when the current
destination is nil or cannot receive immediately.
