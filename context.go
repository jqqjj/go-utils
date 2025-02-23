package utils

import "context"

func ContextFromChan[T any](ctx context.Context, channels ...chan T) (context.Context, context.CancelFunc) {
	subCtx, subCancel := context.WithCancel(ctx)

	for _, v := range channels {
		go func(v chan T) {
			defer subCancel()
			select {
			case <-subCtx.Done():
			case <-v:
			}
		}(v)
	}

	return subCtx, subCancel
}
