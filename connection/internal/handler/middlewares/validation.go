package middlewares

import (
	"connection/internal/handler"
	"errors"
	"fmt"
)

// ValidationMiddleware validates event input before calling next.
func ValidationMiddleware(v handler.Validator) handler.Middleware {
	return func(next handler.HandlerFunc) handler.HandlerFunc {
		return func(c *handler.Context) error {
			if c == nil {
				return errors.New("context is required")
			}
			if v == nil {
				return errors.New("validator is required")
			}
			if err := v.Validate(c); err != nil {
				return fmt.Errorf("event validation failed: %w", err)
			}
			return next(c)
		}
	}
}
