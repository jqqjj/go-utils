package utils

import (
	"context"
	"sync"
)

type Exchanger struct {
	mux         sync.RWMutex
	subscribers map[string][]struct {
		ctx context.Context
		ch  chan<- any
	}
}

func NewExchanger() *Exchanger {
	return &Exchanger{
		subscribers: make(map[string][]struct {
			ctx context.Context
			ch  chan<- any
		}),
	}
}

func (e *Exchanger) Subscribe(ctx context.Context, topic string, ch chan<- any) {
	e.mux.Lock()
	defer e.mux.Unlock()

	e.subscribers[topic] = append(e.subscribers[topic], struct {
		ctx context.Context
		ch  chan<- any
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

func (e *Exchanger) Publish(topic string, data interface{}) {
	var (
		ok         bool
		collection []struct {
			ctx context.Context
			ch  chan<- any
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
			go func(ctx context.Context, ch chan<- any) {
				select {
				case <-ctx.Done():
				case ch <- data:
				}
			}(v.ctx, v.ch)
		}
	}
}
