package sse

import (
	"testing"
	"time"
)

type testEvent struct {
	Type string
	ID   string
}

func TestEventBus_SubscribeReceivesEvents(t *testing.T) {
	bus := NewEventBus[testEvent]()
	ch := bus.Subscribe("client1")
	defer bus.Unsubscribe("client1")

	go func() {
		bus.Publish(testEvent{Type: "image", ID: "img-1"})
	}()

	select {
	case ev := <-ch:
		if ev.Type != "image" {
			t.Errorf("expected type image, got %s", ev.Type)
		}
		if ev.ID != "img-1" {
			t.Errorf("expected ID img-1, got %s", ev.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestEventBus_UnsubscribeStopsEvents(t *testing.T) {
	bus := NewEventBus[testEvent]()
	ch := bus.Subscribe("client1")
	bus.Unsubscribe("client1")

	bus.Publish(testEvent{Type: "image", ID: "img-1"})

	ev, ok := <-ch
	if ok {
		t.Fatal("expected channel to be closed after unsubscribe")
	}
	if ev.Type != "" {
		t.Errorf("expected zero value from closed channel, got %s", ev.Type)
	}
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewEventBus[testEvent]()
	ch1 := bus.Subscribe("client1")
	ch2 := bus.Subscribe("client2")
	defer bus.Unsubscribe("client1")
	defer bus.Unsubscribe("client2")

	bus.Publish(testEvent{Type: "image", ID: "img-1"})

	for _, ch := range []<-chan testEvent{ch1, ch2} {
		select {
		case ev := <-ch:
			if ev.ID != "img-1" {
				t.Errorf("expected img-1, got %s", ev.ID)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out")
		}
	}
}

func TestEventBus_PublishNonBlocking(t *testing.T) {
	bus := NewEventBus[testEvent]()
	_ = bus.Subscribe("slow-client")
	defer bus.Unsubscribe("slow-client")

	done := make(chan struct{})
	go func() {
		for i := 0; i < 20; i++ {
			bus.Publish(testEvent{Type: "image", ID: "img"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publish blocked on slow subscriber")
	}
}
