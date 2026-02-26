package handler

// HandlerChain composes middlewares around a final handler.
type HandlerChain struct {
	final       HandlerFunc
	middlewares []Middleware
}

// NewHandlerChain creates a handler chain around the final handler.
func NewHandlerChain(final HandlerFunc, middlewares ...Middleware) *HandlerChain {
	if final == nil {
		panic("final handler is required")
	}

	chain := &HandlerChain{final: final}
	return chain.Use(middlewares...)
}

// Use appends middlewares to the chain in declaration order.
func (c *HandlerChain) Use(middlewares ...Middleware) *HandlerChain {
	if c == nil {
		panic("handler chain is required")
	}

	for _, mw := range middlewares {
		if mw == nil {
			continue
		}
		c.middlewares = append(c.middlewares, mw)
	}
	return c
}

// Build composes middlewares around the final handler.
func (c *HandlerChain) Build() HandlerFunc {
	if c == nil {
		panic("handler chain is required")
	}

	h := c.final
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		h = c.middlewares[i](h)
	}
	return h
}
