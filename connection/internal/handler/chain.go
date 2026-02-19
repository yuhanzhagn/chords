package handler

// Chain composes middlewares around a final handler, last middleware runs first.
func Chain(final HandlerFunc, middlewares ...Middleware) HandlerFunc {
	if final == nil {
		panic("final handler is required")
	}

	h := final
	for i := len(middlewares) - 1; i >= 0; i-- {
		mw := middlewares[i]
		if mw == nil {
			continue
		}
		h = mw(h)
	}
	return h
}
