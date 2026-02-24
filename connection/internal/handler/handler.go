package handler

import (
	"context"
	"errors"
)

// HandlerFunc processes an event context.
type HandlerFunc func(*Context) error

// Middleware wraps a handler with additional behavior.
type Middleware func(next HandlerFunc) HandlerFunc

// SinkFunc writes a fully processed event to its destination.
type SinkFunc func(ctx context.Context, event any) error

// Validator validates an event context before it reaches downstream handlers.
type Validator interface {
	Validate(*Context) error
}

// SinkHandler is the final handler that writes events to a sink.
func SinkHandler(sink SinkFunc) HandlerFunc {
	return func(c *Context) error {
		if c == nil {
			return errors.New("context is required")
		}
		if sink == nil {
			return errors.New("sink is required")
		}
		if c.Context == nil {
			c.Context = context.Background()
		}
		return sink(c.Context, c.Event)
	}
}
