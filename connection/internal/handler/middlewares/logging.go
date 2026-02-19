package middlewares

import (
	"connection/internal/handler"
	"errors"
	"time"
)

// Logger is the minimal logging contract required by LoggingMiddleware.
type Logger interface {
	Printf(format string, args ...any)
}

// LoggingMiddleware logs start/end and latency for each event.
func LoggingMiddleware(logger Logger) handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(c *handler.Context) error {
			if c == nil {
				return errors.New("context is required")
			}
			if c.ReceivedAt.IsZero() {
				c.ReceivedAt = time.Now()
			}

			start := time.Now()
			if logger != nil {
				logger.Printf("event processing started: client_id=%d", c.ClientID)
			}

			err := next(c)

			if logger != nil {
				logger.Printf(
					"event processing finished: client_id=%d duration=%s err=%v",
					c.ClientID,
					time.Since(start),
					err,
				)
			}
			return err
		}
	}
}
