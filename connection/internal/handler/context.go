package handler

import (
	"context"
	"time"
)

// Context carries decoded event data and metadata through the middleware pipeline.
type Context struct {
	Context    context.Context
	ClientID   uint32
	Event      any
	ReceivedAt time.Time
	Reply      func(any) error
	Values     map[string]any
}
