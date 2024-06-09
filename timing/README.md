# timing

timing helper functions.

## CreateWaiterFn variants

In test environment, some conditions are observed via channels and its send / receive unblocking timing in a manner of `happens before`.

CreateWaiterFn and its variant functions call given variadic functions then returns waiter which blocks until all given functions unblock.

For example:

```go
    waiter := timinghelper.CreateWaiterFn(func() { <-someCh })
    // do something that cause someCh to receive.
    waiter()
```

This is useful when you need to block on receive of channels right before doing some works that send on channels, while avoiding race conditions.

## PollUntil

PollUntil calls given predicate multiple times periodically until it finally returns true, or timeout duration passed.
