package utils

import (
	"context"
	"fmt"
	"io"
	"sync"
)

type PipelineQueuer[T any] interface {
	Enqueue(step int, entity T) error
	Dequeue(ctx context.Context, step int) (T, error)
	ACK(step int, entity T) error
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
	if retries < 1 {
		retries = 1
	}
	if worker < 1 {
		worker = 1
	}
	p.handlers = append(p.handlers, pipelineHandler[T]{name: name, worker: worker, retries: retries, fn: fn})
}

func (p *Pipeline[T]) Queue(entity T) error {
	if len(p.handlers) == 0 {
		return fmt.Errorf("no handlers registered")
	}
	return p.queuer.Enqueue(0, entity)
}

func (p *Pipeline[T]) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for i := range p.handlers {
		h := p.handlers[i]
		for w := 0; w < h.worker; w++ {
			wg.Add(1)
			go func(step int, handler pipelineHandler[T]) {
				defer wg.Done()
				p.runWorker(ctx, step, handler)
			}(i, h)
		}
	}

	wg.Wait()
}

func (p *Pipeline[T]) runWorker(ctx context.Context, step int, h pipelineHandler[T]) {
	for {
		entity, err := p.queuer.Dequeue(ctx, step)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
			p.reportErr(fmt.Errorf("dequeue (step: %d, name: %s) failed: %w", step, p.handlers[step].name, err))
			continue
		}

		var nextEntity T
		caller := func(ctx context.Context, ent T) (err error) {
			for w := 0; w < h.retries; w++ {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				if nextEntity, err = h.fn(ctx, ent); err == nil {
					break
				}
			}
			return
		}

		if err = caller(ctx, entity); err != nil {
			select {
			case <-ctx.Done():
				//外部取消时，需要重新入列
				if err = p.queuer.Enqueue(step, entity); err != nil {
					p.reportErr(fmt.Errorf("retry enqueue (step: %d, name: %s) failed: %w", step, p.handlers[step].name, err))
				} else if err = p.queuer.ACK(step, entity); err != nil {
					p.reportErr(fmt.Errorf("ACK (step: %d, name: %s) failed: %w", step, p.handlers[step].name, err))
				}
				return
			default: //失败时不删除ack以供日志回溯
				p.reportErr(fmt.Errorf("handler (step: %d, name: %s) failed after %d retries: %w", step, p.handlers[step].name, h.retries, err))
				continue
			}
		}

		if nextStep := step + 1; nextStep < len(p.handlers) {
			if err = p.queuer.Enqueue(nextStep, nextEntity); err != nil {
				p.reportErr(fmt.Errorf("enqueue next step (step: %d, name: %s) failed: %w", nextStep, p.handlers[nextStep].name, err))
			} else if err = p.queuer.ACK(step, entity); err != nil {
				p.reportErr(fmt.Errorf("ACK (step: %d, name: %s) failed: %w", step, p.handlers[step].name, err))
			}
		} else {
			if err = p.queuer.ACK(step, entity); err != nil {
				p.reportErr(fmt.Errorf("ACK (step: %d, name: %s) failed: %w", step, p.handlers[step].name, err))
			}
		}
	}
}

func (p *Pipeline[T]) reportErr(err error) {
	if p.logWriter != nil {
		_, _ = p.logWriter.Write([]byte(err.Error() + "\n"))
	}
}
