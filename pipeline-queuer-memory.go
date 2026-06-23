package utils

import (
	"context"
	"sync"
)

type PipelineQueuerMemory[T any] struct {
	mu       sync.Mutex
	notifies map[int]chan struct{}
	queues   map[int][]T
}

func NewPipelineQueuerMemory[T any]() *PipelineQueuerMemory[T] {
	return &PipelineQueuerMemory[T]{
		notifies: make(map[int]chan struct{}),
		queues:   make(map[int][]T),
	}
}

func (q *PipelineQueuerMemory[T]) ACK(step int, entity T) error {
	return nil
}

func (q *PipelineQueuerMemory[T]) Enqueue(step int, entity T) error {
	q.mu.Lock()

	if _, ok := q.queues[step]; !ok {
		q.queues[step] = make([]T, 0)
	}
	q.queues[step] = append(q.queues[step], entity)

	q.mu.Unlock()

	select {
	case q.getNotify(step) <- struct{}{}:
	default:
	}
	return nil
}

func (q *PipelineQueuerMemory[T]) Dequeue(ctx context.Context, step int) (entity T, err error) {
	var ok bool
	if entity, ok, err = q.tryDequeue(step); err != nil || ok {
		return
	}

	ch := q.getNotify(step)
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		case <-ch:
		}

		if entity, ok, err = q.tryDequeue(step); err != nil || ok {
			return
		}
	}
}

func (q *PipelineQueuerMemory[T]) tryDequeue(step int) (entity T, ok bool, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok = q.queues[step]; ok && len(q.queues[step]) > 0 {
		entity, ok = q.queues[step][0], true
		q.queues[step] = q.queues[step][1:]
	}
	return
}

func (q *PipelineQueuerMemory[T]) getNotify(step int) chan struct{} {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.notifies[step]; !ok {
		q.notifies[step] = make(chan struct{}, 1)
	}
	return q.notifies[step]
}
