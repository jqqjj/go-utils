package utils

import (
	"context"
	"fmt"
	"io"
	"sync"
)

type PipelineEntity[T any] struct {
	Entity  T
	Attempt int
}

type PipelineQueuer[T any] interface {
	Enqueue(queue string, job PipelineEntity[T]) error
	Dequeue(ctx context.Context, queue string) (PipelineEntity[T], error)
}

type PipelineHandlerFunc[T any] func(ctx context.Context, entity T) (T, error)

type pipelineHandler[T any] struct {
	name    string
	worker  int
	retries int
	fn      PipelineHandlerFunc[T]
}

type Pipeline[T any] struct {
	queuer    PipelineQueuer[T]
	handlers  []pipelineHandler[T]
	logWriter io.Writer
}

func NewPipeline[T any](queuer PipelineQueuer[T]) *Pipeline[T] {
	return &Pipeline[T]{queuer: queuer}
}

func (p *Pipeline[T]) SetLogger(out io.Writer) {
	p.logWriter = out
}

func (p *Pipeline[T]) Register(name string, retries, worker int, fn PipelineHandlerFunc[T]) {
	p.handlers = append(p.handlers, pipelineHandler[T]{name: name, worker: worker, retries: retries, fn: fn})
}

func (p *Pipeline[T]) Queue(entity T) error {
	if len(p.handlers) == 0 {
		return fmt.Errorf("no handlers registered")
	}
	return p.queuer.Enqueue(p.queueName(0), PipelineEntity[T]{Entity: entity})
}

func (p *Pipeline[T]) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for i := range p.handlers {
		h := p.handlers[i]
		if h.worker <= 0 {
			h.worker = 1
		}

		for w := 0; w < h.worker; w++ {
			wg.Add(1)
			go func(stage int, handler pipelineHandler[T]) {
				defer wg.Done()
				p.runWorker(ctx, stage, handler)
			}(i, h)
		}
	}

	wg.Wait()
}

func (p *Pipeline[T]) runWorker(ctx context.Context, stage int, h pipelineHandler[T]) {
	queue := p.queueName(stage)

	for {
		job, err := p.queuer.Dequeue(ctx, queue)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
			p.reportErr(fmt.Errorf("dequeue %s failed: %w", queue, err))
			continue
		}

		nextEntity, err := h.fn(ctx, job.Entity)
		if err != nil {
			select {
			case <-ctx.Done():
				if err = p.queuer.Enqueue(queue, job); err != nil {
					p.reportErr(fmt.Errorf("retry enqueue %s failed: %w", queue, err))
				}
				return
			default:
			}
			if job.Attempt < h.retries {
				job.Attempt++
				if err = p.queuer.Enqueue(queue, job); err != nil {
					p.reportErr(fmt.Errorf("retry enqueue %s failed: %w", queue, err))
				}
			} else {
				p.reportErr(fmt.Errorf("handler %s failed after %d retries: %w", h.name, h.retries, err))
			}
			continue
		}

		if nextStage := stage + 1; nextStage < len(p.handlers) {
			if err = p.queuer.Enqueue(p.queueName(nextStage), PipelineEntity[T]{Entity: nextEntity}); err != nil {
				p.reportErr(fmt.Errorf("enqueue next stage %s failed: %w", p.queueName(nextStage), err))
			}
		}
	}
}

func (p *Pipeline[T]) queueName(stage int) string {
	return fmt.Sprintf("pipeline:stage:%d:%s", stage, p.handlers[stage].name)
}

func (p *Pipeline[T]) reportErr(err error) {
	if p.logWriter != nil {
		_, _ = p.logWriter.Write([]byte(err.Error() + "\n"))
	}
}
