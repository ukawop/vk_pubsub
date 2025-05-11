package subpub

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestSubscribe(t *testing.T) {
	sp := NewSubPub()
	subj, err := sp.Subscribe("svo", func(msg interface{}) {})

	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	if subj == nil {
		t.Error("Expected subscription, got nil")
	}
}

func TestSubscribe_Success(t *testing.T) {
	sp := NewSubPub()
	defer sp.Close(context.Background())

	done := make(chan struct{})

	sub, err := sp.Subscribe("test", func(msg interface{}) {
		close(done)
	})

	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub.Unsubscribe()

	sp.Publish("test", "message")

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Error("Message was not received")
	}
}

func TestPublish_ClosedSystem(t *testing.T) {
	sp := NewSubPub()
	sp.Close(context.Background())

	err := sp.Publish("test", "message")
	if err == nil {
		t.Error("Expected error when publishing to closed system")
	}
}

func TestConcurrentPublish(t *testing.T) {
	sp := NewSubPub()
	defer sp.Close(context.Background())

	var counter int
	var mu sync.Mutex
	var wg sync.WaitGroup

	sub, _ := sp.Subscribe("test", func(msg interface{}) {
		mu.Lock()
		counter++
		mu.Unlock()
		wg.Done()
	})
	defer sub.Unsubscribe()

	const messages = 100
	wg.Add(messages)

	for i := 0; i < messages; i++ {
		go func() {
			sp.Publish("test", "message")
		}()
	}

	wg.Wait()

	if counter != messages {
		t.Errorf("Expected %d messages, got %d", messages, counter)
	}
}

func TestSubscribe_InvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		topic   string
		handler MessageHandler
		wantErr bool
	}{
		{"Empty topic", "", nil, true},
		{"Nil handler", "valid", nil, true},
		{"Valid inputs", "valid", func(msg interface{}) {}, false},
	}

	sp := NewSubPub()
	defer sp.Close(context.Background())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sp.Subscribe(tt.topic, tt.handler)
			if (err != nil) != tt.wantErr {
				t.Errorf("Subscribe() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
