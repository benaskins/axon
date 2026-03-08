package sse

import (
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

// Compile-time check: both EventBus and NATSEventBus satisfy Publisher.
var (
	_ Publisher[testEvent] = (*EventBus[testEvent])(nil)
	_ Publisher[testEvent] = (*NATSEventBus[testEvent])(nil)
)

func natsConn(t *testing.T) *nats.Conn {
	t.Helper()
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = nats.DefaultURL
	}
	nc, err := nats.Connect(url, nats.Timeout(2*time.Second))
	if err != nil {
		t.Skipf("skipping: NATS not available at %s: %v", url, err)
	}
	t.Cleanup(nc.Close)
	return nc
}

func TestNATSEventBus_SubscribeReceivesEvents(t *testing.T) {
	nc := natsConn(t)
	bus := NewNATSEventBus[testEvent](nc, WithSubject("test.subscribe"))
	t.Cleanup(bus.Close)

	ch := bus.Subscribe("client1")
	defer bus.Unsubscribe("client1")

	// Allow NATS subscription to be established
	nc.Flush()

	bus.Publish(testEvent{Type: "image", ID: "img-1"})
	nc.Flush()

	select {
	case ev := <-ch:
		if ev.Type != "image" {
			t.Errorf("expected type image, got %s", ev.Type)
		}
		if ev.ID != "img-1" {
			t.Errorf("expected ID img-1, got %s", ev.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestNATSEventBus_UnsubscribeStopsEvents(t *testing.T) {
	nc := natsConn(t)
	bus := NewNATSEventBus[testEvent](nc, WithSubject("test.unsub"))
	t.Cleanup(bus.Close)

	ch := bus.Subscribe("client1")
	bus.Unsubscribe("client1")

	ev, ok := <-ch
	if ok {
		t.Fatal("expected channel to be closed after unsubscribe")
	}
	if ev.Type != "" {
		t.Errorf("expected zero value from closed channel, got %s", ev.Type)
	}
}

func TestNATSEventBus_MultipleSubscribers(t *testing.T) {
	nc := natsConn(t)
	bus := NewNATSEventBus[testEvent](nc, WithSubject("test.multi"))
	t.Cleanup(bus.Close)

	ch1 := bus.Subscribe("client1")
	ch2 := bus.Subscribe("client2")
	defer bus.Unsubscribe("client1")
	defer bus.Unsubscribe("client2")

	nc.Flush()

	bus.Publish(testEvent{Type: "image", ID: "img-1"})
	nc.Flush()

	for _, ch := range []<-chan testEvent{ch1, ch2} {
		select {
		case ev := <-ch:
			if ev.ID != "img-1" {
				t.Errorf("expected img-1, got %s", ev.ID)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out")
		}
	}
}

func TestNATSEventBus_CrossInstance(t *testing.T) {
	nc := natsConn(t)

	// Simulate two service instances sharing a NATS connection
	// (in production these would be separate connections to the same cluster)
	bus1 := NewNATSEventBus[testEvent](nc, WithSubject("test.cross"))
	bus2 := NewNATSEventBus[testEvent](nc, WithSubject("test.cross"))
	t.Cleanup(bus1.Close)
	t.Cleanup(bus2.Close)

	ch := bus2.Subscribe("client-on-instance-2")
	defer bus2.Unsubscribe("client-on-instance-2")

	nc.Flush()

	// Publish on instance 1, expect delivery on instance 2
	bus1.Publish(testEvent{Type: "update", ID: "u-1"})
	nc.Flush()

	select {
	case ev := <-ch:
		if ev.Type != "update" || ev.ID != "u-1" {
			t.Errorf("unexpected event: %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for cross-instance event")
	}
}

func TestNATSEventBus_PublishNonBlocking(t *testing.T) {
	nc := natsConn(t)
	bus := NewNATSEventBus[testEvent](nc, WithSubject("test.nonblock"))
	t.Cleanup(bus.Close)

	_ = bus.Subscribe("slow-client")
	defer bus.Unsubscribe("slow-client")

	nc.Flush()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 20; i++ {
			bus.Publish(testEvent{Type: "image", ID: "img"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publish blocked on slow subscriber")
	}
}

func TestNATSEventBus_WithSubject(t *testing.T) {
	nc := natsConn(t)
	bus := NewNATSEventBus[testEvent](nc, WithSubject("custom.subject"))
	t.Cleanup(bus.Close)

	ch := bus.Subscribe("client1")
	defer bus.Unsubscribe("client1")
	nc.Flush()

	bus.Publish(testEvent{Type: "test", ID: "t-1"})
	nc.Flush()

	select {
	case ev := <-ch:
		if ev.Type != "test" {
			t.Errorf("expected type test, got %s", ev.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}
}
