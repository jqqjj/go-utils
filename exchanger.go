package utils

import (
	"context"
	"sync"
)

type Exchanger[T any] struct {
	mux         sync.RWMutex
	subscribers map[string][]struct {
		ctx context.Context
		ch  chan<- T
	}
}

func NewExchanger[T any]() *Exchanger[T] {
	return &Exchanger[T]{
		subscribers: make(map[string][]struct {
			ctx context.Context
			ch  chan<- T
		}),
	}
}

func (e *Exchanger[T]) Subscribe(ctx context.Context, topic string, ch chan<- T) {
	e.mux.Lock()
	defer e.mux.Unlock()

	e.subscribers[topic] = append(e.subscribers[topic], struct {
		ctx context.Context
		ch  chan<- T
	}{ctx: ctx, ch: ch})

	go func() {
		<-ctx.Done()

		e.mux.Lock()
		defer e.mux.Unlock()

		index := -1
		for i, v := range e.subscribers[topic] {
			if v.ch == ch {
				index = i
				break
			}
		}
		if index > -1 {
			copy(e.subscribers[topic][index:], e.subscribers[topic][index+1:])
			e.subscribers[topic] = e.subscribers[topic][:len(e.subscribers[topic])-1]
		}
		if len(e.subscribers[topic]) == 0 {
			delete(e.subscribers, topic)
		}
	}()
}

func (e *Exchanger[T]) Publish(topic string, data T) {
	var (
		ok         bool
		collection []struct {
			ctx context.Context
			ch  chan<- T
		}
	)

	e.mux.RLock()
	defer e.mux.RUnlock()

	if collection, ok = e.subscribers[topic]; !ok {
		return
	}

	for _, v := range collection {
		select {
		case <-v.ctx.Done():
			continue
		case v.ch <- data:
		default:
			go func(ctx context.Context, ch chan<- T) {
				select {
				case <-ctx.Done():
				case ch <- data:
				}
			}(v.ctx, v.ch)
		}
	}
}

func (e *Exchanger[T]) TopicCount() int {
	return len(e.subscribers)
}

func (e *Exchanger[T]) SubscriberCountOfTopic(topic string) int {
	e.mux.RLock()
	defer e.mux.RUnlock()

	if _, ok := e.subscribers[topic]; ok {
		return len(e.subscribers[topic])
	}
	return 0
}
