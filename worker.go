package utils

import (
	"context"
	"sync/atomic"
)

type WorkerPool struct {
	total int
	idle  int32

	pubSub *PubSub[string, any]
}

func NewWorkerPool(num int) *WorkerPool {
	return &WorkerPool{
		total: num,
		idle:  int32(num),

		pubSub: NewPubSub[string, any](),
	}
}

func (w *WorkerPool) Submit(ctx context.Context, fn func(ctx context.Context)) chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

		subCtx, cancelFn := context.WithCancel(ctx)
		defer cancelFn()

		chSub := make(chan any)
		w.pubSub.Subscribe(subCtx, "idle", chSub)

		for {
			select {
			case <-subCtx.Done():
				return
			default:
			}

			if w.fetchToken() {
				fn(subCtx)
				w.freeToken()
				return
			}

			select {
			case <-subCtx.Done():
				return
			case <-chSub:
			}
		}
	}()

	return done
}

func (w *WorkerPool) fetchToken() bool {
	old := w.idle
	if old <= 0 {
		return false
	}
	return atomic.CompareAndSwapInt32(&w.idle, old, old-1)
}

func (w *WorkerPool) freeToken() {
	atomic.AddInt32(&w.idle, 1)
	w.pubSub.Publish("idle", struct{}{})
}

func (w *WorkerPool) SetWorkerNum(num int) {
	if num < 0 {
		return
	}
	offset := num - w.total
	w.total = num
	atomic.AddInt32(&w.idle, int32(offset))
	if offset > 0 {
		w.pubSub.Publish("idle", struct{}{})
	}
}
