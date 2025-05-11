package subpub

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

// MessageHandler is a callback function that processes messages
type MessageHandler func(msg interface{})

type Subscription interface {
	Unsubscribe()
}

type SubPub interface {
	Subscribe(subject string, cb MessageHandler) (Subscription, error)
	Publish(subject string, msg interface{}) error
	Close(ctx context.Context) error
}

type subscription struct {
	subject    string
	handler    MessageHandler
	msgChan    chan interface{}
	doneChan   chan struct{}
	sp         *subPub
	handlerCtx context.Context
}

type subPub struct {
	mu         sync.RWMutex
	subjects   map[string][]*subscription
	closed     bool
	closeChan  chan struct{}
	shardCount int
	shards     []*subPubShard
}

type subPubShard struct {
	mu       sync.RWMutex
	subjects map[string][]*subscription
}

const ( // TODO вынести в конфиг
	defaultBufferSize = 100
	defaultShardCount = 32
	handlerTimeout    = 5 * time.Second
)

func NewSubPub() SubPub {
	sp := &subPub{
		subjects:   make(map[string][]*subscription),
		closeChan:  make(chan struct{}),
		shardCount: defaultShardCount,
		shards:     make([]*subPubShard, defaultShardCount),
	}

	for i := range defaultShardCount {
		sp.shards[i] = &subPubShard{
			subjects: make(map[string][]*subscription),
		}
	}

	return sp
}

func (s *subscription) Unsubscribe() {
	s.sp.mu.Lock()
	defer s.sp.mu.Unlock()

	if s.sp.closed {
		return
	}

	shard := s.sp.getShard(s.subject)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	subs := shard.subjects[s.subject]
	for i, sub := range subs {
		if sub == s {
			copy(subs[i:], subs[i+1:])
			shard.subjects[s.subject] = subs[:len(subs)-1]
			close(s.doneChan)
			return
		}
	}
}

func (sp *subPub) getShard(subject string) *subPubShard {
	// Simple hash-based sharding
	h := fnv32(subject)
	return sp.shards[h%uint32(sp.shardCount)]
}

func fnv32(s string) uint32 {
	h := uint32(2166136261)
	for _, c := range s {
		h *= 16777619
		h ^= uint32(c)
	}
	return h
}

func (sp *subPub) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	if subject == "" {
		return nil, errors.New("subject cannot be empty")
	}
	if cb == nil {
		return nil, errors.New("handler cannot be nil")
	}

	sp.mu.RLock()
	if sp.closed {
		sp.mu.RUnlock()
		return nil, errors.New("subpub is closed")
	}
	sp.mu.RUnlock()

	sub := &subscription{
		subject:    subject,
		handler:    cb,
		msgChan:    make(chan interface{}, defaultBufferSize),
		doneChan:   make(chan struct{}),
		sp:         sp,
		handlerCtx: context.Background(),
	}

	shard := sp.getShard(subject)
	shard.mu.Lock()
	shard.subjects[subject] = append(shard.subjects[subject], sub)
	shard.mu.Unlock()

	go sub.processMessages()

	return sub, nil
}

func (s *subscription) processMessages() {
	defer close(s.msgChan)

	for {
		select {
		case msg, ok := <-s.msgChan:
			if !ok {
				return
			}

			func() {
				ctx, cancel := context.WithTimeout(s.handlerCtx, handlerTimeout)
				defer cancel()

				done := make(chan struct{})
				go func() {
					defer close(done)
					s.handler(msg)
				}()

				select {
				case <-done:
				case <-ctx.Done():
					log.Printf("handler timed out for subject %s", s.subject)
				}
			}()

		case <-s.doneChan:
			return
		case <-s.sp.closeChan:
			return
		}
	}
}

func (sp *subPub) Publish(subject string, msg interface{}) error {
	if subject == "" {
		return errors.New("subject cannot be empty")
	}

	sp.mu.RLock()
	defer sp.mu.RUnlock()

	if sp.closed {
		return errors.New("subpub is closed")
	}

	shard := sp.getShard(subject)
	shard.mu.RLock()
	subs := make([]*subscription, len(shard.subjects[subject]))
	copy(subs, shard.subjects[subject])
	shard.mu.RUnlock()

	var publishErr error

	for _, sub := range subs {
		select {
		case sub.msgChan <- msg:
		default:
			if publishErr == nil {
				publishErr = errors.New("some subscribers dropped messages due to full buffer")
			}
		}
	}

	return publishErr
}

func (sp *subPub) Close(ctx context.Context) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.closed {
		return nil
	}

	sp.closed = true
	close(sp.closeChan)

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
