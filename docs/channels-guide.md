# How Channels Are Used in Axon

Axon uses Go channels in three distinct areas, each demonstrating a different
concurrency pattern. This guide walks through each one.

---

## 1. Graceful Shutdown — Signal Notification

**File:** `server.go`

`ListenAndServe` needs to block the main goroutine until an OS signal
(SIGINT or SIGTERM) arrives, then shut down cleanly. Go's `signal.NotifyContext`
wraps this pattern — it returns a context whose `Done()` channel closes when
the signal fires:

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

go func() {
    // Start HTTP server in a background goroutine
    srv.ListenAndServe()
}()

<-ctx.Done()                // blocks until SIGINT/SIGTERM
slog.Info("shutting down...")
```

`<-ctx.Done()` is a channel receive. The main goroutine parks here doing
nothing until the OS delivers a signal, at which point the channel closes and
execution continues into the two-phase shutdown (hooks, then drain).

**Key takeaway:** You don't need to create a raw `chan os.Signal` yourself —
`signal.NotifyContext` gives you a context-based API that plays well with
timeouts and cancellation throughout the shutdown sequence.

---

## 2. Background Task Control — Stop Channel

**File:** `auth.go`

`AuthClient` runs a background goroutine that periodically sweeps expired
entries from its session cache. The goroutine needs to run indefinitely but
also stop cleanly when the client is closed. A `chan struct{}` serves as the
stop signal:

```go
type AuthClient struct {
    // ...
    closeOnce sync.Once
    stopSweep chan struct{}
}
```

The channel is created during construction:

```go
stopSweep: make(chan struct{})
```

The background goroutine uses `select` to multiplex between the ticker
(periodic work) and the stop signal:

```go
func (ac *AuthClient) sweepExpired() {
    ticker := time.NewTicker(ac.config.cacheTTL)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            // Walk the cache, delete expired entries
            ac.cache.Range(func(key, value any) bool {
                // ...expiry check...
                return true
            })
        case <-ac.stopSweep:
            return
        }
    }
}
```

Closing the channel signals the goroutine to exit:

```go
func (ac *AuthClient) Close() {
    ac.closeOnce.Do(func() {
        close(ac.stopSweep)
    })
}
```

Three things to note:

- **`chan struct{}`** carries no data — it's purely a signal. Closing it
  unblocks every receiver, which is exactly what you want for "stop now."
- **`sync.Once`** protects against double-close panics. Closing an already-closed
  channel panics in Go, so `closeOnce` makes `Close()` safe to call multiple
  times.
- **`select`** lets the goroutine respond to whichever event arrives first —
  either the next tick or the stop signal.

---

## 3. Pub/Sub Event Distribution — EventBus

**File:** `sse/eventbus.go`

`EventBus[T]` is the most channel-intensive part of Axon. It implements
in-memory fan-out for Server-Sent Events: one publisher, many subscribers,
each with their own buffered channel.

### The data structure

```go
type EventBus[T any] struct {
    mu   sync.Mutex
    subs map[string]chan T
}
```

Each subscriber is identified by a client ID and gets a dedicated channel.

### Subscribe — buffered channels and directional types

```go
func (b *EventBus[T]) Subscribe(clientID string) <-chan T {
    b.mu.Lock()
    defer b.mu.Unlock()
    ch := make(chan T, 16)
    b.subs[clientID] = ch
    return ch
}
```

Two design choices here:

- **Buffer of 16** — the subscriber doesn't have to be reading at the exact
  moment an event is published. The buffer absorbs short bursts. Without a
  buffer, every publish would block until the subscriber reads.
- **Return type `<-chan T`** — the caller gets a receive-only channel. They can
  read from it but can't close it or send into it. The EventBus retains the
  bidirectional `chan T` internally so only it can send and close.

### Unsubscribe — closing channels

```go
func (b *EventBus[T]) Unsubscribe(clientID string) {
    b.mu.Lock()
    defer b.mu.Unlock()
    if ch, ok := b.subs[clientID]; ok {
        close(ch)
        delete(b.subs, clientID)
    }
}
```

Closing the channel tells the subscriber "no more events." A subscriber
reading with `for ev := range ch` will exit the loop naturally. A subscriber
using `ev, ok := <-ch` will see `ok == false`.

### Publish — snapshot + non-blocking send

```go
func (b *EventBus[T]) Publish(ev T) {
    b.mu.Lock()
    snapshot := make(map[string]chan T, len(b.subs))
    for id, ch := range b.subs {
        snapshot[id] = ch
    }
    b.mu.Unlock()

    for id, ch := range snapshot {
        select {
        case ch <- ev:
        default:
            slog.Warn("event bus: dropping event for slow subscriber", "client", id)
        }
    }
}
```

This is the most interesting method. Two patterns work together:

1. **Snapshot before sending** — the mutex is held only long enough to copy the
   subscriber map. Sends happen outside the lock. This prevents a slow
   subscriber from blocking all other subscribers (and any subscribe/unsubscribe
   calls).

2. **Non-blocking send with `select`/`default`** — if a subscriber's buffer is
   full (they're not reading fast enough), the event is dropped rather than
   blocking the publisher. The `default` branch in a `select` runs immediately
   when no other case can proceed. This is critical for SSE: a stalled browser
   tab shouldn't back up the entire event pipeline.

---

## Patterns in Tests

The test files demonstrate a few more channel idioms worth knowing:

### Done channels for goroutine coordination

```go
done := make(chan struct{})
go func() {
    axon.ListenAndServe("0", mux)
    close(done)
}()

// ... trigger shutdown ...

select {
case <-done:
    // server exited
case <-time.After(5 * time.Second):
    t.Fatal("server did not shut down within 5 seconds")
}
```

The `done` channel lets the test wait for a goroutine to finish. Combined with
`select` and `time.After`, it becomes a timeout — the test fails rather than
hanging forever.

### Buffered channels to collect results

```go
orderCh := make(chan []int, 1)
go func() {
    // ... collect results ...
    orderCh <- results
}()
```

A buffered channel of capacity 1 lets the goroutine send its result and exit
without waiting for the receiver. This avoids a goroutine leak if the test
times out and never reads from the channel.

---

## Summary

| Pattern | Where | Channel Type | Why |
|---|---|---|---|
| Signal wait | `server.go` | `context.Done()` (unbuffered) | Park main goroutine until OS signal |
| Stop signal | `auth.go` | `chan struct{}` (unbuffered) | Tell background goroutine to exit |
| Pub/sub fan-out | `sse/eventbus.go` | `chan T` (buffered, cap 16) | Deliver events to SSE subscribers |
| Non-blocking send | `sse/eventbus.go` | `select`/`default` | Drop events for slow subscribers |
| Done channel | tests | `chan struct{}` (unbuffered) | Wait for goroutine completion with timeout |
| Result channel | tests | `chan []int` (buffered, cap 1) | Collect goroutine output without blocking |
