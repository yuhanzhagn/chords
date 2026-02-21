package sink

import "context"

// Sink dispatches values to an external system.
type Sink[T any] interface {
	Write(ctx context.Context, value T) error
}
