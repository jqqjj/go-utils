package utils

import (
	"context"
	"sync"
)

type PipelineQueuerMemory[T any] struct {
	mu       sync.Mutex
	notifies map[string]chan struct{}
	queues   map[string][]PipelineEntity[T]
}

func NewPipelineQueuerMemory[T any]() *PipelineQueuerMemory[T] {
	return &PipelineQueuerMemory[T]{
		notifies: make(map[string]chan struct{}),
		queues:   make(map[string][]PipelineEntity[T]),
	}
}

func (q *PipelineQueuerMemory[T]) Enqueue(queue string, job PipelineEntity[T]) error {
	q.mu.Lock()

	if _, ok := q.queues[queue]; !ok {
		q.queues[queue] = make([]PipelineEntity[T], 0)
	}
	q.queues[queue] = append(q.queues[queue], job)

	q.mu.Unlock()

	q.notify(queue)
	return nil
}

func (q *PipelineQueuerMemory[T]) Dequeue(ctx context.Context, queue string) (job PipelineEntity[T], err error) {
	var ok bool
	if job, ok, err = q.tryDequeue(queue); err != nil {
		return
	}
	if ok {
		return job, nil
	}

	ch := q.getNotify(queue)
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		case <-ch:
		}

		if job, ok, err = q.tryDequeue(queue); err != nil {
			return
		}
		if ok {
			return job, nil
		}
	}
}

func (q *PipelineQueuerMemory[T]) tryDequeue(queue string) (job PipelineEntity[T], ok bool, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok = q.queues[queue]; ok && len(q.queues[queue]) > 0 {
		job = q.queues[queue][0]
		ok = true
		q.queues[queue] = q.queues[queue][1:]
	}

	return
}

func (q *PipelineQueuerMemory[T]) getNotify(queue string) chan struct{} {
	q.mu.Lock()
	defer q.mu.Unlock()

	ch, ok := q.notifies[queue]
	if !ok {
		ch = make(chan struct{}, 1)
		q.notifies[queue] = ch
	}
	return ch
}

func (q *PipelineQueuerMemory[T]) notify(queue string) {
	select {
	case q.getNotify(queue) <- struct{}{}:
	default:
	}
}
