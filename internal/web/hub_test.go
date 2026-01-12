package web

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHub_Broadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Mock client 1
	client1 := &Client{
		hub:  hub,
		send: make(chan []byte, 256),
	}
	hub.register <- client1

	// Mock client 2
	client2 := &Client{
		hub:  hub,
		send: make(chan []byte, 256),
	}
	hub.register <- client2

	// Wait for registration
	time.Sleep(10 * time.Millisecond)

	// Broadcast message
	msg := map[string]string{"type": "job_update", "status": "new"}
	msgBytes, _ := json.Marshal(msg)
	hub.broadcast <- msgBytes

	// Verify clients received message
	select {
	case received := <-client1.send:
		assert.Equal(t, msgBytes, received)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Client 1 did not receive message")
	}

	select {
	case received := <-client2.send:
		assert.Equal(t, msgBytes, received)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Client 2 did not receive message")
	}

	// Unregister client 1
	hub.unregister <- client1
	time.Sleep(10 * time.Millisecond)

	// Broadcast another message
	msg2 := []byte("second message")
	hub.broadcast <- msg2

	// Client 1 should NOT receive it (channel closed or nothing sent)
	select {
	case msg, ok := <-client1.send:
		if ok {
			t.Fatalf("Client 1 received message after unregister: %s", msg)
		}
		// if !ok, channel is closed, which is correct behavior for unregistered client
	case <-time.After(50 * time.Millisecond):
		// Success
	}

	// Client 2 SHOULD receive it
	select {
	case received := <-client2.send:
		assert.Equal(t, msg2, received)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Client 2 did not receive second message")
	}
}
