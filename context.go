package utils

import "context"

func contextFromChan[T any](ctx context.Context, ch chan T) (context.Context, context.CancelFunc) {
	subCtx, subCancel := context.WithCancel(ctx)

	go func() {
		defer subCancel()

		select {
		case <-subCtx.Done():
		case <-ch:
		}
	}()

	return subCtx, subCancel
}
