package sse

import (
	"log/slog"
	"sync"
)

// EventBus is a generic in-memory pub/sub for delivering async events to SSE clients.
type EventBus[T any] struct {
	mu   sync.Mutex
	subs map[string]chan T
}

func NewEventBus[T any]() *EventBus[T] {
	return &EventBus[T]{subs: make(map[string]chan T)}
}

func (b *EventBus[T]) Subscribe(clientID string) <-chan T {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan T, 16)
	b.subs[clientID] = ch
	return ch
}

func (b *EventBus[T]) Unsubscribe(clientID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ch, ok := b.subs[clientID]; ok {
		close(ch)
		delete(b.subs, clientID)
	}
}

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
