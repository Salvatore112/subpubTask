package subpub

import (
	"context"
	"sync"
)

// MessageHandler is a callback function that processes messages delivered to subscribers.
type MessageHandler func(msg interface{})

type Subscription interface {
	// Unsubscribe will remove interest in the current subject subscription is for.
	Unsubscribe()
}

// SubPub is the main interface for the publish-subscribe system
type SubPub interface {
	// Subscribe creates an asynchronous queue subscriber on the given subject.
	Subscribe(subject string, cb MessageHandler) (Subscription, error)
	// Publish publishes the msg argument to the given subject.
	Publish(subject string, msg interface{}) error
	// Close will shutdown sub-pub system.
	// May be blocked by data delivery until the context is canceled.
	Close(ctx context.Context) error
}

type subscription struct {
	subject  string
	handler  MessageHandler
	msgChan  chan interface{}
	bus      *subPub
	cancel   context.CancelFunc
	isActive bool
	mu       sync.Mutex
}

func (s *subscription) Unsubscribe() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isActive {
		return
	}

	s.isActive = false
	s.bus.removeSubscription(s)
	s.cancel()
	close(s.msgChan)
}

type subPub struct {
	subscribers map[string][]*subscription
	mu          sync.RWMutex
	wg          sync.WaitGroup
	closed      bool
}

func NewSubPub() SubPub {
	return &subPub{
		subscribers: make(map[string][]*subscription),
	}
}

func (b *subPub) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, context.Canceled
	}

	ctx, cancel := context.WithCancel(context.Background())
	msgChan := make(chan interface{}, 100)

	sub := &subscription{
		subject:  subject,
		handler:  cb,
		msgChan:  msgChan,
		bus:      b,
		cancel:   cancel,
		isActive: true,
	}

	b.subscribers[subject] = append(b.subscribers[subject], sub)

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		for {
			select {
			case msg, ok := <-msgChan:
				if !ok {
					return
				}
				cb(msg)
			case <-ctx.Done():
				return
			}
		}
	}()

	return sub, nil
}

func (b *subPub) Publish(subject string, msg interface{}) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return context.Canceled
	}

	subs, ok := b.subscribers[subject]
	if !ok {
		return nil
	}

	for _, sub := range subs {
		sub.mu.Lock()
		if sub.isActive {
			sub.msgChan <- msg
		}
		sub.mu.Unlock()
	}

	return nil
}

func (b *subPub) Close(ctx context.Context) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()

	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *subPub) removeSubscription(sub *subscription) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs, ok := b.subscribers[sub.subject]
	if !ok {
		return
	}

	for i, s := range subs {
		if s == sub {
			b.subscribers[sub.subject] = append(subs[:i], subs[i+1:]...)
			break
		}
	}

	if len(b.subscribers[sub.subject]) == 0 {
		delete(b.subscribers, sub.subject)
	}
}
