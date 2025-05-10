package subpub

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSubscribePublish(t *testing.T) {
	bus := NewSubPub()
	var wg sync.WaitGroup

	received := make(chan string, 1)
	handler := func(msg interface{}) {
		received <- msg.(string)
		wg.Done()
	}

	sub, err := bus.Subscribe("test", handler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub.Unsubscribe()

	wg.Add(1)
	err = bus.Publish("test", "hello")
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	select {
	case msg := <-received:
		if msg != "hello" {
			t.Errorf("Expected 'hello', got '%s'", msg)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for message")
	}

	wg.Wait()
}

func TestUnsubscribe(t *testing.T) {
	bus := NewSubPub()
	received := make(chan string, 1)
	handler := func(msg interface{}) {
		received <- msg.(string)
	}

	sub, err := bus.Subscribe("test", handler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	sub.Unsubscribe()

	err = bus.Publish("test", "hello")
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	select {
	case <-received:
		t.Error("Received message after unsubscribe")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

func TestSlowSubscriber(t *testing.T) {
	bus := NewSubPub()
	var wg sync.WaitGroup

	fastReceived := make(chan string, 10)
	slowReceived := make(chan string, 10)

	fastHandler := func(msg interface{}) {
		fastReceived <- msg.(string)
		wg.Done()
	}

	slowHandler := func(msg interface{}) {
		time.Sleep(100 * time.Millisecond)
		slowReceived <- msg.(string)
		wg.Done()
	}

	sub1, err := bus.Subscribe("test", fastHandler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub1.Unsubscribe()

	sub2, err := bus.Subscribe("test", slowHandler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub2.Unsubscribe()

	wg.Add(4)
	for i := 0; i < 2; i++ {
		err = bus.Publish("test", "hello")
		if err != nil {
			t.Fatalf("Publish failed: %v", err)
		}
	}

	// Verify fast subscriber got messages quickly
	select {
	case msg := <-fastReceived:
		if msg != "hello" {
			t.Errorf("Expected 'hello', got '%s'", msg)
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("Fast subscriber didn't receive message quickly")
	}

	wg.Wait()
}

func TestClose(t *testing.T) {
	bus := NewSubPub()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := bus.Close(ctx)
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Test publishing after close
	err = bus.Publish("test", "hello")
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	// Test subscribing after close
	_, err = bus.Subscribe("test", func(msg interface{}) {})
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestOrderPreservation(t *testing.T) {
	bus := NewSubPub()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var received []string

	handler := func(msg interface{}) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, msg.(string))
		wg.Done()
	}

	sub, err := bus.Subscribe("test", handler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub.Unsubscribe()

	wg.Add(3)
	err = bus.Publish("test", "msg1")
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	err = bus.Publish("test", "msg2")
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	err = bus.Publish("test", "msg3")
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	wg.Wait()

	if len(received) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(received))
	}
	if received[0] != "msg1" || received[1] != "msg2" || received[2] != "msg3" {
		t.Errorf("Messages out of order: %v", received)
	}
}
