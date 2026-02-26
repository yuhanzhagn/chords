package handler

import (
	"reflect"
	"testing"
)

func TestHandlerChainBuild_ComposesMiddlewaresInDeclarationOrder(t *testing.T) {
	var steps []string
	final := func(_ *Context) error {
		steps = append(steps, "final")
		return nil
	}

	mwA := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			steps = append(steps, "mwA-before")
			err := next(c)
			steps = append(steps, "mwA-after")
			return err
		}
	}
	mwB := func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			steps = append(steps, "mwB-before")
			err := next(c)
			steps = append(steps, "mwB-after")
			return err
		}
	}

	h := NewHandlerChain(final, mwA, mwB).Build()
	if err := h(&Context{}); err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}

	want := []string{"mwA-before", "mwB-before", "final", "mwB-after", "mwA-after"}
	if !reflect.DeepEqual(steps, want) {
		t.Fatalf("unexpected middleware order: got %v want %v", steps, want)
	}
}

func TestHandlerChainBuild_SkipsNilMiddlewares(t *testing.T) {
	finalCalls := 0
	final := func(_ *Context) error {
		finalCalls++
		return nil
	}

	var nilMiddleware Middleware
	h := NewHandlerChain(final, nilMiddleware).Build()
	if err := h(&Context{}); err != nil {
		t.Fatalf("unexpected handler error: %v", err)
	}
	if finalCalls != 1 {
		t.Fatalf("expected final handler to be called once, got %d", finalCalls)
	}
}

func TestNewHandlerChain_PanicsOnNilFinalHandler(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when final handler is nil")
		}
	}()

	NewHandlerChain(nil)
}
