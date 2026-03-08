package sse

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/nats-io/nats.go"
)

// NATSEventBus is a distributed pub/sub backed by NATS, enabling horizontal
// scaling of SSE services. Events published on any instance are delivered to
// subscribers on all instances connected to the same NATS cluster.
type NATSEventBus[T any] struct {
	conn    *nats.Conn
	subject string

	mu   sync.Mutex
	subs map[string]*natsClient[T]
}

type natsClient[T any] struct {
	ch   chan T
	nsub *nats.Subscription
}

// NATSEventBusOption configures a NATSEventBus.
type NATSEventBusOption func(*natsEventBusConfig)

type natsEventBusConfig struct {
	subject string
}

// WithSubject sets the NATS subject used for publishing and subscribing.
// Defaults to "events".
func WithSubject(subject string) NATSEventBusOption {
	return func(c *natsEventBusConfig) {
		c.subject = subject
	}
}

// NewNATSEventBus creates a NATSEventBus connected to the given NATS connection.
// Each subscriber gets a unique NATS subscription on the configured subject,
// ensuring fan-out delivery across all instances.
func NewNATSEventBus[T any](conn *nats.Conn, opts ...NATSEventBusOption) *NATSEventBus[T] {
	cfg := natsEventBusConfig{subject: "events"}
	for _, o := range opts {
		o(&cfg)
	}
	return &NATSEventBus[T]{
		conn:    conn,
		subject: cfg.subject,
		subs:    make(map[string]*natsClient[T]),
	}
}

func (b *NATSEventBus[T]) Subscribe(clientID string) <-chan T {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan T, 16)

	nsub, err := b.conn.Subscribe(b.subject, func(msg *nats.Msg) {
		var ev T
		if err := json.Unmarshal(msg.Data, &ev); err != nil {
			slog.Warn("nats event bus: failed to unmarshal event",
				"client", clientID, "error", err)
			return
		}
		select {
		case ch <- ev:
		default:
			slog.Warn("nats event bus: dropping event for slow subscriber",
				"client", clientID)
		}
	})
	if err != nil {
		slog.Error("nats event bus: failed to subscribe",
			"client", clientID, "error", err)
		close(ch)
		return ch
	}

	b.subs[clientID] = &natsClient[T]{ch: ch, nsub: nsub}
	return ch
}

func (b *NATSEventBus[T]) Unsubscribe(clientID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	c, ok := b.subs[clientID]
	if !ok {
		return
	}

	_ = c.nsub.Unsubscribe()
	close(c.ch)
	delete(b.subs, clientID)
}

func (b *NATSEventBus[T]) Publish(ev T) {
	data, err := json.Marshal(ev)
	if err != nil {
		slog.Error("nats event bus: failed to marshal event", "error", err)
		return
	}

	if err := b.conn.Publish(b.subject, data); err != nil {
		slog.Error("nats event bus: failed to publish", "error", err)
	}
}

// Close drains all subscriptions and closes local channels.
func (b *NATSEventBus[T]) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for id, c := range b.subs {
		_ = c.nsub.Unsubscribe()
		close(c.ch)
		delete(b.subs, id)
	}
}
