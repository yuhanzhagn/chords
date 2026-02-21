package sink

import (
	"errors"
	"fmt"
	"strings"
)

// MultiError preserves all errors produced by a fan-out operation.
type MultiError struct {
	errs []error
}

// NewMultiError builds a MultiError from a list, dropping nil values.
func NewMultiError(errs ...error) error {
	filtered := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			filtered = append(filtered, err)
		}
	}

	switch len(filtered) {
	case 0:
		return nil
	case 1:
		return filtered[0]
	default:
		return &MultiError{errs: filtered}
	}
}

func (m *MultiError) Error() string {
	if m == nil || len(m.errs) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("multiple errors: ")
	for i, err := range m.errs {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(fmt.Sprintf("[%d] %v", i, err))
	}
	return b.String()
}

// Unwrap returns all underlying errors, enabling errors.Is/errors.As.
func (m *MultiError) Unwrap() []error {
	if m == nil {
		return nil
	}
	out := make([]error, 0, len(m.errs))
	for _, err := range m.errs {
		if err != nil {
			out = append(out, err)
		}
	}
	return out
}

// Errors returns a copy of underlying errors.
func (m *MultiError) Errors() []error {
	return m.Unwrap()
}

// Has reports whether MultiError contains target using errors.Is semantics.
func (m *MultiError) Has(target error) bool {
	if m == nil {
		return false
	}
	return errors.Is(m, target)
}
