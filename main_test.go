package main

import (
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	client := new(Client)

	if err := client.Connection(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	client.WriteMessagesToChannel()

	if err := client.SubscribeToChannel("BTC_USDT"); err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	ch := make(chan BestOrderBook)

	client.ReadMessagesFromChannel(ch)

	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("Failed to read message")
	}

	client.Disconnect()
}
